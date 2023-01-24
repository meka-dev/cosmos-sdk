package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdk_x_staking_types "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// BankKeeper defines the contract needed to be fulfilled for banking and supply
// dependencies.
type BankKeeper interface {
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	SendCoins(ctx sdk.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
}

// AccountKeeper defines the contract required for account APIs.
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) sdk.AccountI
	GetModuleAccount(ctx sdk.Context, moduleName string) sdk.ModuleAccountI
}

// StakingKeeper defines the contract required for staking APIs.
type StakingKeeper interface {
	GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator sdk_x_staking_types.Validator, found bool)
	GetLastValidators(ctx sdk.Context) (validators []sdk_x_staking_types.Validator)
	PowerReduction(ctx sdk.Context) math.Int
}
