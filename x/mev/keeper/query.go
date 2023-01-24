package keeper

import (
	sdk_x_builder_types "cosmossdk.io/x/mev/types"
)

var _ sdk_x_builder_types.QueryServer = Keeper{}
