package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/types/msgservice"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgAddOracle{}, "icaoracle/AddOracle", nil)
	cdc.RegisterConcrete(&MsgInstantiateOracle{}, "icaoracle/InstantiateOracle", nil)
	cdc.RegisterConcrete(&MsgRestoreOracleICA{}, "icaoracle/RestoreOracleICA", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAddOracle{},
		&MsgRestoreOracleICA{},
	)

	registry.RegisterImplementations((*govtypes.Content)(nil),
		&ToggleOracleProposal{},
		&RemoveOracleProposal{},
		&UpdateOracleContractProposal{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)