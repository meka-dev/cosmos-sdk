package keeper

import (
	"context"

	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	sdk_types_errors "github.com/cosmos/cosmos-sdk/types/errors"
	sdk_x_gov_types "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// UpdateParams implements MsgServer.UpdateParams method.
// It defines a method to update the x/mev module parameters.
func (k msgServer) UpdateParams(goCtx context.Context, req *sdk_x_builder_types.MsgUpdateParams) (*sdk_x_builder_types.MsgUpdateParamsResponse, error) {
	if k.authority != req.Authority {
		return nil, sdk_types_errors.Wrapf(sdk_x_gov_types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	ctx := sdk_types.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, sdk_types_errors.Wrapf(sdk_types_errors.ErrInvalidRequest, "set params: %v", err)
	}

	return &sdk_x_builder_types.MsgUpdateParamsResponse{}, nil
}
