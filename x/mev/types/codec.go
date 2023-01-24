package types

import (
	"sync"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterBuilder{}, "cosmos-sdk/x/mev/RegisterBuilder", nil)
	cdc.RegisterConcrete(&MsgEditBuilder{}, "cosmos-sdk/x/mev/EditBuilder", nil)
	cdc.RegisterConcrete(&MsgRegisterProposer{}, "cosmos-sdk/x/mev/RegisterProposer", nil)
	cdc.RegisterConcrete(&MsgCommitSegment{}, "cosmos-sdk/x/mev/CommitSegment", nil)
	cdc.RegisterConcrete(&MsgReportProposer{}, "cosmos-sdk/x/mev/ReportProposer", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "cosmos-sdk/x/mev/MsgUpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterBuilder{},
		&MsgEditBuilder{},
		&MsgRegisterProposer{},
		&MsgCommitSegment{},
		&MsgReportProposer{},
		&MsgUpdateParams{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	once      sync.Once
	moduleCdc *codec.ProtoCodec
)

// ModuleCdc returns the codec.ProtoCodec with registered interfaces.
// It's implemented as a method to ensure the init functions in the same
// package in generated proto code run before.
func ModuleCdc() *codec.ProtoCodec {
	once.Do(func() {
		r := cdctypes.NewInterfaceRegistry()
		std.RegisterInterfaces(r)
		RegisterInterfaces(r)
		moduleCdc = codec.NewProtoCodec(r)
	})
	return moduleCdc
}
