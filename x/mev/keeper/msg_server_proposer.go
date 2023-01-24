package keeper

import (
	"context"

	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_crypto_types "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	sdk_types_errors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) RegisterProposer(goCtx context.Context, msg *sdk_x_builder_types.MsgRegisterProposer) (*sdk_x_builder_types.MsgRegisterProposerResponse, error) {
	ctx := sdk_types.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, sdk_types_errors.Wrap(err, "register proposer")
	}

	if _, isFound := k.GetProposer(ctx, msg.Address); isFound {
		return nil, sdk_types_errors.Wrap(sdk_types_errors.ErrInvalidRequest, "proposer with given address already registered")
	}

	pubKey, ok := msg.Pubkey.GetCachedValue().(sdk_crypto_types.PubKey)
	if !ok {
		return nil, sdk_types_errors.Wrapf(sdk_types_errors.ErrInvalidType, "PubKey must be sdk_crypto_types.PubKey, got %T", pubKey)
	}

	if have, want := msg.Address, sdk_types.AccAddress(pubKey.Address()).String(); have != want {
		return nil, sdk_types_errors.Wrapf(sdk_x_builder_types.ErrAddressPubKeyMismatch, "address: have %q, want %q", have, want)
	}

	operatorPubKey, ok := msg.OperatorPubkey.GetCachedValue().(sdk_crypto_types.PubKey)
	if !ok {
		return nil, sdk_types_errors.Wrapf(sdk_types_errors.ErrInvalidType, "operatorPubKey must be cryptotypes.PubKey, got %T", operatorPubKey)
	}

	operatorAddr := sdk_types.ValAddress(operatorPubKey.Address())
	if have, want := msg.OperatorAddress, operatorAddr.String(); have != want {
		return nil, sdk_types_errors.Wrapf(sdk_x_builder_types.ErrAddressPubKeyMismatch, "operator address: have %q, want %q", have, want)
	}

	if _, isFound := k.sk.GetValidator(ctx, operatorAddr); !isFound {
		return nil, sdk_types_errors.Wrap(sdk_types_errors.ErrInvalidRequest, "validator with given address not found")
	}

	// TODO: We need to verify the the operator owns this builder module pub key. Can we leverage multi-sig here by requiring
	// the message to be signed with both keys?

	k.SetProposer(ctx, sdk_x_builder_types.Proposer{
		Address:         msg.Address,
		Pubkey:          msg.Pubkey,
		OperatorAddress: msg.OperatorAddress,
		OperatorPubkey:  msg.OperatorPubkey,
	})

	return &sdk_x_builder_types.MsgRegisterProposerResponse{}, nil
}
