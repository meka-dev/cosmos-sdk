package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk_crypto_types "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TypeMsgRegisterProposer = "register_proposer"
)

var _ sdk.Msg = &MsgRegisterProposer{}

func NewMsgRegisterProposer(
	address sdk.AccAddress,
	pubKey cryptotypes.PubKey, //nolint:interfacer
	operatorAddress sdk.ValAddress,
	operatorPubKey cryptotypes.PubKey,
) (*MsgRegisterProposer, error) {
	msg := &MsgRegisterProposer{
		Address:         address.String(),
		OperatorAddress: operatorAddress.String(),
	}

	if pubKey != nil {
		var err error
		if msg.Pubkey, err = codectypes.NewAnyWithValue(pubKey); err != nil {
			return nil, err
		}
	}

	if pubKey != nil {
		var err error
		if msg.OperatorPubkey, err = codectypes.NewAnyWithValue(pubKey); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

func (msg *MsgRegisterProposer) Route() string {
	return RouterKey
}

func (msg *MsgRegisterProposer) Type() string {
	return TypeMsgRegisterProposer
}

func (msg *MsgRegisterProposer) GetSigners() []sdk.AccAddress {
	operatorAddress, err := sdk.ValAddressFromBech32(msg.OperatorAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{sdk.AccAddress(operatorAddress)}
}

func (msg *MsgRegisterProposer) GetSignBytes() []byte {
	bz := ModuleCdc().MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgRegisterProposer) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address (%s)", err)
	}

	if msg.Pubkey == nil {
		return ErrEmptyPubKey
	}

	_, err = sdk.ValAddressFromBech32(msg.OperatorAddress)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid operator address (%s)", err)
	}

	if msg.OperatorPubkey == nil {
		return ErrEmptyOperatorPubKey
	}

	return nil
}

func (msg *MsgRegisterProposer) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pubKey sdk_crypto_types.PubKey
	if err := unpacker.UnpackAny(msg.Pubkey, &pubKey); err != nil {
		return err
	}
	if err := unpacker.UnpackAny(msg.OperatorPubkey, &pubKey); err != nil {
		return err
	}
	return nil
}
