package mev

import (
	"fmt"

	"cosmossdk.io/x/mev/keeper"
	"cosmossdk.io/x/mev/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)

	moduleAcc := k.GetBuilderModuleAccount(ctx)
	if moduleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	// Set all the builder
	for _, elem := range genState.Builders {
		k.SetBuilder(ctx, elem)
	}

	// Set all the proposer
	for _, elem := range genState.Proposers {
		k.SetProposer(ctx, elem)
	}
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()

	genesis.Params = k.GetParams(ctx)
	genesis.Builders = k.GetBuilders(ctx)
	genesis.Proposers = k.GetProposers(ctx)

	return genesis
}
