package keeper

import (
	sdk_store_prefix "cosmossdk.io/store/prefix"
	sdk_store_types "cosmossdk.io/store/types"
	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
)

// SetSegmentCommitment set a builder commitment in the store.
func (k Keeper) SetSegmentCommitment(ctx sdk_types.Context, sc sdk_x_builder_types.SegmentCommitment) {
	store := sdk_store_prefix.NewStore(
		ctx.KVStore(k.storeKey),
		sdk_x_builder_types.KeyPrefix(
			sdk_x_builder_types.SegmentCommitmentStoreKeyPrefix,
		),
	)
	store.Set(sdk_x_builder_types.SegmentCommitmentStoreKey(
		sc.SignaturesHash(),
	), k.cdc.MustMarshal(&sc))
}

// GetSegmentCommitment returns a builder commitment by its signatures.
func (k Keeper) GetSegmentCommitment(ctx sdk_types.Context, signaturesHash []byte) (sdk_x_builder_types.SegmentCommitment, bool) {
	store := sdk_store_prefix.NewStore(
		ctx.KVStore(k.storeKey),
		sdk_x_builder_types.KeyPrefix(
			sdk_x_builder_types.SegmentCommitmentStoreKeyPrefix,
		),
	)

	data := store.Get(sdk_x_builder_types.SegmentCommitmentStoreKey(
		signaturesHash,
	))

	if data == nil {
		return sdk_x_builder_types.SegmentCommitment{}, false
	}

	var val sdk_x_builder_types.SegmentCommitment
	if err := k.cdc.Unmarshal(data, &val); err != nil {
		ctx.Logger().Error("error unmarshaling segment commitment", "err", err)
		return sdk_x_builder_types.SegmentCommitment{}, false
	}

	return val, true
}

// GetSegmentCommitment returns a builder commitment by height.
func (k Keeper) GetSegmentCommitmentByHeight(ctx sdk_types.Context, height int64) (sdk_x_builder_types.SegmentCommitment, bool) {
	kvStore := ctx.KVStore(k.storeKey)
	store := sdk_store_prefix.NewStore(
		kvStore,
		sdk_x_builder_types.KeyPrefix(
			sdk_x_builder_types.SegmentCommitmentStoreKeyPrefix,
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

		if sc.Height == height {
			return sc, true
		}
	}

	return sdk_x_builder_types.SegmentCommitment{}, false
}

// DeleteOldSegmentCommitments deletes any segment commitments that have height less than minHeight.
func (k Keeper) DeleteOldSegmentCommitments(ctx sdk_types.Context, minHeight int64) {
	kvStore := ctx.KVStore(k.storeKey)
	store := sdk_store_prefix.NewStore(
		kvStore,
		sdk_x_builder_types.KeyPrefix(
			sdk_x_builder_types.SegmentCommitmentStoreKeyPrefix,
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
