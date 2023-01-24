package keeper

import (
	"context"
	"fmt"

	sdk_store_prefix "cosmossdk.io/store/prefix"
	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	sdk_types_query "github.com/cosmos/cosmos-sdk/types/query"
	sdk_x_staking_types "github.com/cosmos/cosmos-sdk/x/staking/types"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Proposers(goCtx context.Context, req *sdk_x_builder_types.QueryProposersRequest) (*sdk_x_builder_types.QueryProposersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var proposers []sdk_x_builder_types.Proposer
	ctx := sdk_types.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	proposerStore := sdk_store_prefix.NewStore(store, sdk_x_builder_types.KeyPrefix(sdk_x_builder_types.ProposerStoreKeyPrefix))

	pageRes, err := sdk_types_query.Paginate(proposerStore, req.Pagination, func(key []byte, value []byte) error {
		var val sdk_x_builder_types.Proposer
		if err := k.cdc.Unmarshal(value, &val); err != nil {
			return err
		}
		proposers = append(proposers, val)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &sdk_x_builder_types.QueryProposersResponse{Proposers: proposers, Pagination: pageRes}, nil
}

func (k Keeper) Proposer(goCtx context.Context, req *sdk_x_builder_types.QueryProposerRequest) (*sdk_x_builder_types.QueryProposerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk_types.UnwrapSDKContext(goCtx)

	proposer, ok := k.GetProposer(ctx, req.Address)
	if !ok {
		return nil, status.Error(codes.NotFound, "not found")
	}

	valAddr, err := sdk_types.ValAddressFromBech32(proposer.OperatorAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("operator address %s is not a val address: %v", proposer.OperatorAddress, err))
	}

	var validator *sdk_x_staking_types.Validator
	if v, ok := k.sk.GetValidator(ctx, valAddr); ok {
		validator = &v
	}

	resp := &sdk_x_builder_types.QueryProposerResponse{
		Proposer:  proposer,
		Validator: validator,
	}

	if req.Infractions {
		resp.Infractions = k.GetProposerInfractions(ctx, req.Address)
	}

	return resp, nil
}
