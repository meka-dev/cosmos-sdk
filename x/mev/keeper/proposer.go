package keeper

import (
	sdk_store_prefix "cosmossdk.io/store/prefix"
	sdk_store_types "cosmossdk.io/store/types"
	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
)

// SetProposer set a specific proposer in the store from its index
func (k Keeper) SetProposer(ctx sdk_types.Context, proposer sdk_x_builder_types.Proposer) {
	kvStore := ctx.KVStore(k.storeKey)

	store := sdk_store_prefix.NewStore(kvStore, sdk_x_builder_types.KeyPrefix(sdk_x_builder_types.ProposerStoreKeyPrefix))
	store.Set(sdk_x_builder_types.ProposerStoreKey(proposer.Address), k.cdc.MustMarshal(&proposer))

	index := sdk_store_prefix.NewStore(kvStore, sdk_x_builder_types.KeyPrefix(sdk_x_builder_types.ProposerStoreOperatorAddressIndexPrefix))
	index.Set(sdk_x_builder_types.ProposerByOperatorAddressKey(proposer.OperatorAddress), []byte(proposer.Address))
}

// GetProposer returns a proposer from its address
func (k Keeper) GetProposer(ctx sdk_types.Context, address string) (sdk_x_builder_types.Proposer, bool) {
	store := sdk_store_prefix.NewStore(ctx.KVStore(k.storeKey), sdk_x_builder_types.KeyPrefix(sdk_x_builder_types.ProposerStoreKeyPrefix))

	data := store.Get(sdk_x_builder_types.ProposerStoreKey(address))

	if data == nil {
		return sdk_x_builder_types.Proposer{}, false
	}

	var val sdk_x_builder_types.Proposer
	if err := k.cdc.Unmarshal(data, &val); err != nil {
		ctx.Logger().Error("error unmarshaling proposer", "err", err)
		return sdk_x_builder_types.Proposer{}, false
	}

	return val, true
}

// GetProposerByOperatorAddress returns a proposer from its operator address.
func (k Keeper) GetProposerByOperatorAddress(ctx sdk_types.Context, operatorAddress string) (sdk_x_builder_types.Proposer, bool) {
	index := sdk_store_prefix.NewStore(ctx.KVStore(k.storeKey), sdk_x_builder_types.KeyPrefix(sdk_x_builder_types.ProposerStoreOperatorAddressIndexPrefix))
	address := index.Get(sdk_x_builder_types.ProposerByOperatorAddressKey(operatorAddress))
	return k.GetProposer(ctx, string(address))
}

// GetProposers returns all proposer
func (k Keeper) GetProposers(ctx sdk_types.Context) []sdk_x_builder_types.Proposer {
	store := sdk_store_prefix.NewStore(ctx.KVStore(k.storeKey), sdk_x_builder_types.KeyPrefix(sdk_x_builder_types.ProposerStoreKeyPrefix))
	iterator := sdk_store_types.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	var list []sdk_x_builder_types.Proposer
	for ; iterator.Valid(); iterator.Next() {
		data := iterator.Value()

		var val sdk_x_builder_types.Proposer
		if err := k.cdc.Unmarshal(data, &val); err != nil {
			ctx.Logger().Error("error unmarshaling proposer", "err", err)
			continue
		}
		list = append(list, val)
	}

	return list
}
