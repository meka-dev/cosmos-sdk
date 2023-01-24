package keeper

import (
	"fmt"

	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
)

// SetParams sets the x/mev module parameters.
func (k Keeper) SetParams(ctx sdk_types.Context, params sdk_x_builder_types.Params) error {
	if err := params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&params)
	store.Set(sdk_x_builder_types.ParamsKey, bz)

	return nil
}

// GetParams sets the x/mev module parameters.
func (k Keeper) GetParams(ctx sdk_types.Context) (params sdk_x_builder_types.Params) {
	store := ctx.KVStore(k.storeKey)
	if bz := store.Get(sdk_x_builder_types.ParamsKey); bz != nil {
		k.cdc.MustUnmarshal(bz, &params)
	}
	return params
}
