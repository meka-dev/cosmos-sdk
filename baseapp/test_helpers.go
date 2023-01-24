package baseapp

import (
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// SimCheck defines a CheckTx helper function that used in tests and simulations.
func (app *BaseApp) SimCheck(txEncoder sdk.TxEncoder, tx sdk.Tx) (sdk.GasInfo, *sdk.Result, error) {
	// runTx expects tx bytes as argument, so we encode the tx argument into
	// bytes. Note that runTx will actually decode those bytes again. But since
	// this helper is only used in tests/simulation, it's fine.
	bz, err := txEncoder(tx)
	if err != nil {
		return sdk.GasInfo{}, nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "%s", err)
	}
	gasInfo, result, _, _, err := app.runTx(runTxModeCheck, bz)
	return gasInfo, result, err
}

// Simulate executes a tx in simulate mode to get result and gas info.
func (app *BaseApp) Simulate(txBytes []byte) (sdk.GasInfo, *sdk.Result, error) {
	gasInfo, result, _, _, err := app.runTx(runTxModeSimulate, txBytes)
	return gasInfo, result, err
}

// SimulatePrepareProposal executes a tx in the runTxPrepareProposal. It's used
// by implementations of sdk.PrepareProposalHandler to validate external transactions.
func (app *BaseApp) SimulatePrepareProposal(txBytes []byte) (sdk.GasInfo, *sdk.Result, error) {
	gasInfo, result, _, _, err := app.runTx(runTxPrepareProposal, txBytes)
	return gasInfo, result, err
}

// SimulateProcessProposal executes a tx in the runTxProcessProposal. It's used
// by implementations of sdk.ProcessProposalHandler to validate external transactions.
func (app *BaseApp) SimulateProcessProposal(txBytes []byte) (sdk.GasInfo, *sdk.Result, error) {
	gasInfo, result, _, _, err := app.runTx(runTxProcessProposal, txBytes)
	return gasInfo, result, err
}

func (app *BaseApp) SimDeliver(txEncoder sdk.TxEncoder, tx sdk.Tx) (sdk.GasInfo, *sdk.Result, error) {
	// See comment for Check().
	bz, err := txEncoder(tx)
	if err != nil {
		return sdk.GasInfo{}, nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "%s", err)
	}
	gasInfo, result, _, _, err := app.runTx(runTxModeDeliver, bz)
	return gasInfo, result, err
}

// Context with current {check, deliver}State of the app used by tests.
func (app *BaseApp) NewContext(isCheckTx bool, header cmtproto.Header) sdk.Context {
	if isCheckTx {
		return sdk.NewContext(app.checkState.ms, header, true, app.logger).
			WithMinGasPrices(app.minGasPrices)
	}

	return sdk.NewContext(app.deliverState.ms, header, false, app.logger)
}

func (app *BaseApp) NewUncachedContext(isCheckTx bool, header cmtproto.Header) sdk.Context {
	return sdk.NewContext(app.cms, header, isCheckTx, app.logger)
}

func (app *BaseApp) GetContextForDeliverTx(txBytes []byte) sdk.Context {
	return app.getContextForTx(runTxModeDeliver, txBytes)
}
