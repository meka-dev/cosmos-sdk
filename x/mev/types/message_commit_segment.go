package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgCommitSegment = "commit_segment"

var _ sdk.Msg = &MsgCommitSegment{}

func (msg *MsgCommitSegment) Route() string {
	return RouterKey
}

func (msg *MsgCommitSegment) Type() string {
	return TypeMsgCommitSegment
}

func (msg *MsgCommitSegment) GetSigners() []sdk.AccAddress {
	builderAddress, err := sdk.AccAddressFromBech32(msg.BuilderAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{builderAddress}
}

func (msg *MsgCommitSegment) GetSignBytes() []byte {
	bz := ModuleCdc().MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCommitSegment) ValidateBasic() error {
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
