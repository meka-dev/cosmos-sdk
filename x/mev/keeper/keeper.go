package keeper

import (
	sdk_store_types "cosmossdk.io/store/types"
	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_codec "github.com/cosmos/cosmos-sdk/codec"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
)

type (
	Keeper struct {
		cdc      sdk_codec.BinaryCodec
		storeKey sdk_store_types.StoreKey
		//paramstore  sdk_x_params_types.Subspace
		ak        sdk_x_builder_types.AccountKeeper
		bk        sdk_x_builder_types.BankKeeper
		sk        sdk_x_builder_types.StakingKeeper
		authority string
	}
)

func NewKeeper(
	cdc sdk_codec.BinaryCodec,
	storeKey sdk_store_types.StoreKey,
	ak sdk_x_builder_types.AccountKeeper,
	bk sdk_x_builder_types.BankKeeper,
	sk sdk_x_builder_types.StakingKeeper,
	authority string,
) Keeper {
	if addr := ak.GetModuleAddress(sdk_x_builder_types.ModuleName); addr == nil {
		panic("the builder module account has not been set")
	}

	return Keeper{
		cdc:       cdc,
		storeKey:  storeKey,
		ak:        ak,
		bk:        bk,
		sk:        sk,
		authority: authority,
	}
}

func (k Keeper) GetBuilderModuleAccount(ctx sdk_types.Context) sdk_types.ModuleAccountI {
	return k.ak.GetModuleAccount(ctx, sdk_x_builder_types.ModuleName)
}
