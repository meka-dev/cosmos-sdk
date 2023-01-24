package mev

import (
	"strconv"
	"time"

	tm_abci_types "github.com/cometbft/cometbft/abci/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
)

func (am *AppModule) ProcessProposal(sdkctx sdk_types.Context, req tm_abci_types.RequestProcessProposal) tm_abci_types.ResponseProcessProposal {
	logger := sdkctx.Logger().With("module", "x/mev", "method", "ProcessProposal", "height", strconv.Itoa(int(req.GetHeight())))
	logger.Debug("starting")
	defer func(begin time.Time) { logger.Debug("finished", "took", time.Since(begin)) }(time.Now())

	logger.Debug("validating block", "tx_count", len(req.GetTxs()))

	if err := am.validateCommitment(validateCommitmentConfig{
		BlockTxs:     req.GetTxs(),
		Required:     false,
		SimulateFunc: am.simulateProcessFunc,
	}); err != nil {
		logger.Error("error validating proposal txs", "err", err)
		return tm_abci_types.ResponseProcessProposal{Status: tm_abci_types.ResponseProcessProposal_REJECT}
	}

	return tm_abci_types.ResponseProcessProposal{Status: tm_abci_types.ResponseProcessProposal_ACCEPT}
}
