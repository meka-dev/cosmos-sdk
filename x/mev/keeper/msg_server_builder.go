package keeper

import (
	"context"

	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	sdk_types_errors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) RegisterBuilder(goCtx context.Context, msg *sdk_x_builder_types.MsgRegisterBuilder) (*sdk_x_builder_types.MsgRegisterBuilderResponse, error) {
	ctx := sdk_types.UnwrapSDKContext(goCtx)

	// Check if the value already exists
	_, isFound := k.GetBuilder(ctx, msg.Address)
	if isFound {
		return nil, sdk_types_errors.Wrap(sdk_types_errors.ErrInvalidRequest, "address already set")
	}

	builder := sdk_x_builder_types.Builder{
		Address:           msg.Address,
		Pubkey:            msg.Pubkey,
		Moniker:           msg.Moniker,
		BuilderApiVersion: msg.BuilderApiVersion,
		BuilderApiUrl:     msg.BuilderApiUrl,
		SecurityContact:   msg.SecurityContact,
	}

	k.SetBuilder(ctx, builder)

	return &sdk_x_builder_types.MsgRegisterBuilderResponse{}, nil
}

func (k msgServer) EditBuilder(goCtx context.Context, msg *sdk_x_builder_types.MsgEditBuilder) (*sdk_x_builder_types.MsgEditBuilderResponse, error) {
	ctx := sdk_types.UnwrapSDKContext(goCtx)

	ctx.Logger().Debug("EditBuilder", "builder_api_url", msg.BuilderApiUrl)

	// Check if the value exists
	builder, isFound := k.GetBuilder(ctx, msg.Address)
	if !isFound {
		ctx.Logger().Debug("EditBuilder error", "err", "address not set", "address", msg.Address)
		return nil, sdk_types_errors.Wrap(sdk_types_errors.ErrKeyNotFound, "address not set")
	}

	builder.Moniker = msg.Moniker
	builder.BuilderApiVersion = msg.BuilderApiVersion
	builder.BuilderApiUrl = msg.BuilderApiUrl
	builder.SecurityContact = msg.SecurityContact

	k.SetBuilder(ctx, builder)

	return &sdk_x_builder_types.MsgEditBuilderResponse{}, nil
}
