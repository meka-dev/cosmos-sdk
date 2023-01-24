package keeper

import (
	"context"

	sdk_store_prefix "cosmossdk.io/store/prefix"
	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	sdk_types_query "github.com/cosmos/cosmos-sdk/types/query"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Builders(goCtx context.Context, req *sdk_x_builder_types.QueryBuildersRequest) (*sdk_x_builder_types.QueryBuildersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var builders []sdk_x_builder_types.Builder
	ctx := sdk_types.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	builderStore := sdk_store_prefix.NewStore(store, sdk_x_builder_types.KeyPrefix(sdk_x_builder_types.BuilderStoreKeyPrefix))

	pageRes, err := sdk_types_query.Paginate(builderStore, req.Pagination, func(key []byte, value []byte) error {
		var val sdk_x_builder_types.Builder
		if err := k.cdc.Unmarshal(value, &val); err != nil {
			return err
		}
		builders = append(builders, val)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &sdk_x_builder_types.QueryBuildersResponse{Builders: builders, Pagination: pageRes}, nil
}

func (k Keeper) Builder(goCtx context.Context, req *sdk_x_builder_types.QueryBuilderRequest) (*sdk_x_builder_types.QueryBuilderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk_types.UnwrapSDKContext(goCtx)

	val, found := k.GetBuilder(ctx, req.Address)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &sdk_x_builder_types.QueryBuilderResponse{Builder: val}, nil
}
