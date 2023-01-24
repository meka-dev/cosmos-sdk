package keeper

import (
	sdk_x_builder_types "cosmossdk.io/x/mev/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) sdk_x_builder_types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ sdk_x_builder_types.MsgServer = msgServer{}
