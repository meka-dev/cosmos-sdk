package keeper

import (
	sdk_store_prefix "cosmossdk.io/store/prefix"
	sdk_store_types "cosmossdk.io/store/types"
	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
)

// SetProposerInfraction stores a proposer's infraction of a segment commitment in the store.
func (k Keeper) SetProposerInfraction(ctx sdk_types.Context, sc sdk_x_builder_types.SegmentCommitment) {
	store := sdk_store_prefix.NewStore(
		ctx.KVStore(k.storeKey),
		sdk_x_builder_types.KeyPrefix(sdk_x_builder_types.ProposerInfractionStoreKeyPrefix),
	)
	store.Set(
		sdk_x_builder_types.ProposerInfractionStoreKey(
			sc.ProposerAddress,
			sc.SignaturesHash(),
		),
		k.cdc.MustMarshal(&sc),
	)
}

// GetProposerInfractions returns all segment commitments that a given proposer has infracted.
func (k Keeper) GetProposerInfractions(ctx sdk_types.Context, proposerAddress string) []*sdk_x_builder_types.SegmentCommitment {
	store := sdk_store_prefix.NewStore(
		ctx.KVStore(k.storeKey),
		sdk_x_builder_types.KeyPrefix(
			sdk_x_builder_types.ProposerInfractionStoreKeyPrefix,
		),
	)
	iterator := sdk_store_types.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	var list []*sdk_x_builder_types.SegmentCommitment
	for ; iterator.Valid(); iterator.Next() {
		data := iterator.Value()

		var val sdk_x_builder_types.SegmentCommitment
		if err := k.cdc.Unmarshal(data, &val); err != nil {
			ctx.Logger().Error("error unmarshaling proposer", "err", err)
			continue
		}
		list = append(list, &val)
	}

	return list
}

// DeleteOldProposerInfractions deletes any proposer infractions that have height less than minHeight.
func (k Keeper) DeleteOldProposerInfractions(ctx sdk_types.Context, minHeight int64) {
	kvStore := ctx.KVStore(k.storeKey)
	store := sdk_store_prefix.NewStore(
		kvStore,
		sdk_x_builder_types.KeyPrefix(
			sdk_x_builder_types.ProposerInfractionStoreKeyPrefix,
		),
	)

	iterator := sdk_store_types.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		data := iterator.Value()

		var sc sdk_x_builder_types.SegmentCommitment
		if err := k.cdc.Unmarshal(data, &sc); err != nil {
			ctx.Logger().Error("error unmarshaling segment commitment", "err", err)
			continue
		}

		if sc.Height < minHeight {
			store.Delete(iterator.Key())
		}
	}
}
