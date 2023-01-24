package types

import (
	"net/url"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TypeMsgRegisterBuilder = "register_builder"
	TypeMsgEditBuilder     = "edit_builder"
)

var _ sdk.Msg = &MsgRegisterBuilder{}

func NewMsgRegisterBuilder(
	address sdk.AccAddress,
	pubKey cryptotypes.PubKey, //nolint:interfacer
	moniker string,
	builderApiVersion string,
	builderApiUrl string,
	securityContact string,
) (*MsgRegisterBuilder, error) {
	msg := &MsgRegisterBuilder{
		Address:           address.String(),
		Moniker:           moniker,
		BuilderApiVersion: builderApiVersion,
		BuilderApiUrl:     builderApiUrl,
		SecurityContact:   securityContact,
	}

	if pubKey != nil {
		var err error
		if msg.Pubkey, err = codectypes.NewAnyWithValue(pubKey); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

func (msg *MsgRegisterBuilder) Route() string {
	return RouterKey
}

func (msg *MsgRegisterBuilder) Type() string {
	return TypeMsgRegisterBuilder
}

func (msg *MsgRegisterBuilder) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgRegisterBuilder) GetSignBytes() []byte {
	bz := ModuleCdc().MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgRegisterBuilder) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address (%s)", err)
	}

	if msg.Pubkey == nil {
		return ErrEmptyPubKey
	}

	if msg.Moniker == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "empty moniker")
	}

	if msg.BuilderApiVersion == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "empty builder API version")
	}

	if msg.BuilderApiUrl == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "empty builder API version")
	}

	_, err = url.Parse(msg.BuilderApiUrl)
	if err != nil {
		return sdkerrors.Wrap(err, "builder API URL invalid")
	}

	if msg.SecurityContact == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "empty security contact")
	}

	return nil
}

var _ sdk.Msg = &MsgEditBuilder{}

func NewMsgEditBuilder(
	address sdk.AccAddress,
	moniker string,
	builderApiVersion string,
	builderApiUrl string,
	securityContact string,
) *MsgRegisterBuilder {
	return &MsgRegisterBuilder{
		Address:           address.String(),
		Moniker:           moniker,
		BuilderApiVersion: builderApiVersion,
		BuilderApiUrl:     builderApiUrl,
		SecurityContact:   securityContact,
	}
}

func (msg *MsgEditBuilder) Route() string {
	return RouterKey
}

func (msg *MsgEditBuilder) Type() string {
	return TypeMsgEditBuilder
}

func (msg *MsgEditBuilder) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgEditBuilder) GetSignBytes() []byte {
	bz := ModuleCdc().MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgEditBuilder) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address (%s)", err)
	}
	return nil
}
