package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgReportProposer = "report_proposer"

var _ sdk.Msg = &MsgReportProposer{}

func (msg *MsgReportProposer) Route() string {
	return RouterKey
}

func (msg *MsgReportProposer) Type() string {
	return TypeMsgReportProposer
}

func (msg *MsgReportProposer) GetSigners() []sdk.AccAddress {
	builderAddress, err := sdk.AccAddressFromBech32(msg.BuilderAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{builderAddress}
}

func (msg *MsgReportProposer) GetSignBytes() []byte {
	bz := ModuleCdc().MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgReportProposer) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.BuilderAddress)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid builder address (%s)", err)
	}

	if msg.BuilderAddress != msg.Commitment.BuilderAddress {
		return sdkerrors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"builder address mismatch (%q != %q)",
			msg.BuilderAddress,
			msg.Commitment.BuilderAddress,
		)
	}

	_, err = sdk.AccAddressFromBech32(msg.Commitment.ProposerAddress)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid proposer address (%s)", err)
	}

	return nil
}
