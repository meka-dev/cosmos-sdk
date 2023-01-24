package keeper

import (
	"context"
	"fmt"

	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_crypto_types "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	sdk_types_errors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) ReportProposer(goCtx context.Context, msg *sdk_x_builder_types.MsgReportProposer) (*sdk_x_builder_types.MsgReportProposerResponse, error) {
	ctx := sdk_types.UnwrapSDKContext(goCtx)

	signers := msg.GetSigners()
	if len(signers) != 1 {
		return nil, fmt.Errorf("%w: commit segment expects a single signer", sdk_types_errors.ErrorInvalidSigner)
	}

	params := k.GetParams(ctx)
	ageBlocks := ctx.BlockHeight() - msg.Commitment.Height
	if ageBlocks > params.MaxEvidenceAgeNumBlocks {
		return nil, sdk_types_errors.Wrapf(sdk_types_errors.ErrInvalidRequest,
			"evidence too old: proposer=%q builder=%q age_blocks=%d max_age_blocks=%d",
			msg.Commitment.ProposerAddress,
			msg.Commitment.BuilderAddress,
			ageBlocks,
			params.MaxEvidenceAgeNumBlocks,
		)
	}

	builderAddr := signers[0].String()
	builder, isFound := k.Keeper.GetBuilder(ctx, builderAddr)
	if !isFound {
		return nil, sdk_types_errors.Wrapf(sdk_types_errors.ErrNotFound, "builder %q not registered", builderAddr)
	}

	// TODO: Verify builder is authorized, not just registered.

	proposer, isFound := k.Keeper.GetProposer(ctx, msg.Commitment.ProposerAddress)
	if !isFound {
		return nil, sdk_types_errors.Wrapf(sdk_types_errors.ErrNotFound, "proposer %q not registered", msg.Commitment.ProposerAddress)
	}

	proposerPubKey := proposer.Pubkey.GetCachedValue().(sdk_crypto_types.PubKey)
	builderPubKey := builder.Pubkey.GetCachedValue().(sdk_crypto_types.PubKey)

	err := msg.Commitment.VerifySignatures(builderPubKey, proposerPubKey)
	if err != nil {
		return nil, sdk_types_errors.Wrapf(sdk_types_errors.ErrInvalidRequest, "invalid segment commitment signatures: %v", err)
	}

	// At this point the signatures of the builder and proposer commitment are verified,
	// so we should have a builder commitment stored. If we don't, the proposer didn't
	// honor its commitment. This is guaranteed by the builder module's state machine via
	// prepare proposal and process proposal.

	_, isFound = k.Keeper.GetSegmentCommitment(ctx, msg.Commitment.SignaturesHash())
	if isFound {
		return nil, sdk_types_errors.Wrapf(sdk_types_errors.ErrInvalidRequest, "segment commitment exists, nothing to report")
	}

	k.Keeper.SetProposerInfraction(ctx, msg.Commitment)

	return &sdk_x_builder_types.MsgReportProposerResponse{}, nil
}
