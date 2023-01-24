package keeper

import (
	"context"
	"fmt"
	"time"

	sdk_errors "cosmossdk.io/errors"
	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_crypto_types "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	sdk_types_errors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) CommitSegment(goCtx context.Context, msg *sdk_x_builder_types.MsgCommitSegment) (_ *sdk_x_builder_types.MsgCommitSegmentResponse, reserr error) {
	sdkctx := sdk_types.UnwrapSDKContext(goCtx)

	logger := sdkctx.Logger().With("module", "x/mev", "method", "CommitSegment", "height", sdkctx.BlockHeight())
	logger.Debug("starting")
	defer func(begin time.Time) { logger.Debug("finished", "err", reserr, "took", time.Since(begin)) }(time.Now())
	sdkctx = sdkctx.WithLogger(logger)

	nextHeight := sdkctx.BlockHeight() // TODO: this was +1, but that wasn't right, I guess the BlockHeight during DeliverTx is the next height already?
	if msg.Commitment.Height != nextHeight {
		return nil, sdk_errors.Wrapf(sdk_types_errors.ErrInvalidRequest, "invalid segment commitment height: %v != %v", msg.Commitment.Height, nextHeight)
	}

	signers := msg.GetSigners()
	if len(signers) != 1 {
		return nil, fmt.Errorf("%w: commit segment expects a single signer", sdk_types_errors.ErrorInvalidSigner)
	}

	builderAddr := signers[0]
	builder, ok := k.Keeper.GetBuilder(sdkctx, builderAddr.String())
	if !ok {
		return nil, sdk_errors.Wrapf(sdk_types_errors.ErrNotFound, "builder %q not registered", builderAddr)
	}

	// TODO: Verify builder is authorized, not just registered.

	// TODO: ProposerForHeight is currently broken
	/*
		// TODO: Factor ProposerForHeight out to GetProposerForHeight which doesn't take a request type in.
		nextHeightProposer, err := k.Keeper.ProposerForHeight(sdkctx, &sdk_x_builder_types.QueryProposerForHeightRequest{Height: nextHeight})
		if err != nil {
			return nil, sdk_errors.Wrapf(sdk_types_errors.ErrLogic, "error fetching proposer for height %d: %v", nextHeight, err)
		}

		if nextHeightProposer.Proposer.Address != msg.Commitment.ProposerAddress {
			return nil, sdk_errors.Wrapf(sdk_types_errors.ErrInvalidRequest, "invalid proposer %q for height %d (want %q)", msg.Commitment.ProposerAddress, nextHeight, nextHeightProposer.Proposer.Address)
		}
	*/

	proposer, isFound := k.Keeper.GetProposer(sdkctx, msg.Commitment.ProposerAddress)
	if !isFound {
		return nil, sdk_errors.Wrapf(sdk_types_errors.ErrNotFound, "proposer %q not registered", msg.Commitment.ProposerAddress)
	}

	proposerPubKey := proposer.Pubkey.GetCachedValue().(sdk_crypto_types.PubKey)
	builderPubKey := builder.Pubkey.GetCachedValue().(sdk_crypto_types.PubKey)
	if err := msg.Commitment.VerifySignatures(builderPubKey, proposerPubKey); err != nil {
		return nil, sdk_errors.Wrapf(sdk_types_errors.ErrInvalidRequest, "invalid segment commitment signatures: %v", err)
	}

	paymentFromAddr := builderAddr
	paymentToAddr := k.ak.GetModuleAddress(sdk_x_builder_types.ModuleName)
	paymentCoin, err := sdk_types.ParseCoinNormalized(msg.Commitment.PaymentPromise) // TODO: use safe helper
	if err != nil {
		return nil, sdk_errors.Wrapf(sdk_types_errors.ErrLogic, "invalid payment promise: %v", err)
	}

	paymentCoins := sdk_types.Coins{paymentCoin}
	if err := k.bk.SendCoins(sdkctx, paymentFromAddr, paymentToAddr, paymentCoins); err != nil {
		return nil, sdk_errors.Wrapf(sdk_types_errors.ErrLogic, "send bid payment: %v", err)
	}

	k.Keeper.SetSegmentCommitment(sdkctx, msg.Commitment)

	return &sdk_x_builder_types.MsgCommitSegmentResponse{}, nil
}
