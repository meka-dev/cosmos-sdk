package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/mev module sentinel errors
var (
	ErrEmptyAddress          = sdkerrors.Register(ModuleName, 1, "empty address")
	ErrEmptyOperatorAddress  = sdkerrors.Register(ModuleName, 2, "empty operator address")
	ErrEmptyPubKey           = sdkerrors.Register(ModuleName, 3, "empty pubkey")
	ErrEmptyOperatorPubKey   = sdkerrors.Register(ModuleName, 4, "empty operator pubkey")
	ErrUnsupportedPubKeyType = sdkerrors.Register(ModuleName, 5, "unsupported pubkey type")
	ErrAddressPubKeyMismatch = sdkerrors.Register(ModuleName, 6, "address doesn't match pubkey")
)
