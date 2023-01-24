package keeper

import (
	"math/rand"

	sdk_store_prefix "cosmossdk.io/store/prefix"
	sdk_store_types "cosmossdk.io/store/types"
	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
)

// SetBuilder set a specific builder in the store from its index
func (k Keeper) SetBuilder(ctx sdk_types.Context, builder sdk_x_builder_types.Builder) {
	store := sdk_store_prefix.NewStore(ctx.KVStore(k.storeKey), sdk_x_builder_types.KeyPrefix(sdk_x_builder_types.BuilderStoreKeyPrefix))
	b := k.cdc.MustMarshal(&builder)
	store.Set(sdk_x_builder_types.BuilderStoreKey(builder.Address), b)
}

// GetBuilder returns a builder from its address
func (k Keeper) GetBuilder(ctx sdk_types.Context, address string) (sdk_x_builder_types.Builder, bool) {
	store := sdk_store_prefix.NewStore(ctx.KVStore(k.storeKey), sdk_x_builder_types.KeyPrefix(sdk_x_builder_types.BuilderStoreKeyPrefix))

	data := store.Get(sdk_x_builder_types.BuilderStoreKey(address))
	if data == nil {
		return sdk_x_builder_types.Builder{}, false
	}

	var val sdk_x_builder_types.Builder
	if err := k.cdc.Unmarshal(data, &val); err != nil {
		ctx.Logger().Error("error unmarshaling builder", "err", err)
		return sdk_x_builder_types.Builder{}, false
	}

	return val, true
}

// RemoveBuilder removes a builder from the store
func (k Keeper) RemoveBuilder(ctx sdk_types.Context, address string) {
	store := sdk_store_prefix.NewStore(ctx.KVStore(k.storeKey), sdk_x_builder_types.KeyPrefix(sdk_x_builder_types.BuilderStoreKeyPrefix))
	store.Delete(sdk_x_builder_types.BuilderStoreKey(address))
}

// GetBuilders returns all builder
func (k Keeper) GetBuilders(ctx sdk_types.Context) []sdk_x_builder_types.Builder {
	return k.getBuilders(ctx, nil)
}

// GetAuctionBuilders returns up to max_builder_per_auction allowed builders, as per module params.
func (k Keeper) GetAuctionBuilders(ctx sdk_types.Context) []sdk_x_builder_types.Builder {
	var (
		pred   func(*sdk_x_builder_types.Builder) bool
		params = k.GetParams(ctx)
	)

	if len(params.AllowedBuilderAddresses) != 0 {
		set := make(map[string]struct{}, len(params.AllowedBuilderAddresses))
		for _, addr := range params.AllowedBuilderAddresses {
			set[addr] = struct{}{}
		}

		pred = func(b *sdk_x_builder_types.Builder) bool {
			_, ok := set[b.Address]
			return ok
		}
	}

	builders := k.getBuilders(ctx, pred)
	rng := rand.New(rand.NewSource(ctx.BlockHeight()))
	rng.Shuffle(len(builders), func(i, j int) {
		builders[i], builders[j] = builders[j], builders[i]
	})

	if int64(len(builders)) <= params.MaxBuildersPerAuction {
		return builders
	}

	return builders[:int(params.MaxBuildersPerAuction)]
}

func (k Keeper) getBuilders(ctx sdk_types.Context, pred func(*sdk_x_builder_types.Builder) bool) []sdk_x_builder_types.Builder {
	store := sdk_store_prefix.NewStore(ctx.KVStore(k.storeKey), sdk_x_builder_types.KeyPrefix(sdk_x_builder_types.BuilderStoreKeyPrefix))
	iterator := sdk_store_types.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	var list []sdk_x_builder_types.Builder
	for ; iterator.Valid(); iterator.Next() {
		data := iterator.Value()

		var val sdk_x_builder_types.Builder
		if err := k.cdc.Unmarshal(data, &val); err != nil {
			ctx.Logger().Error("error unmarshaling builder", "err", err)
			continue
		}

		if pred == nil || pred(&val) {
			list = append(list, val)
		}
	}

	return list
}
