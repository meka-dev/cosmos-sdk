package mev

import (
	"fmt"
	"strconv"
	"time"

	tm_abci_types "github.com/cometbft/cometbft/abci/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	sdk_x_distribution_types "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// PaymentFunc defines how payments from builders are distributed.
//
// It's called at the end of every block, and provides the payment received for
// that block specifically (if there was a successful auction) and the total
// balance of the builder module account. It returns a set of payments, where a
// payment is a recipient address and an amount that address should receive.
//
// Payments are made from the builder module account, and are processed in
// order. Failed payments are logged but otherwise ignored.
type PaymentFunc func(sdkctx sdk_types.Context, auctionPayment sdk_types.Coins, totalBalance sdk_types.Coins) ([]Payment, error)

type Payment struct {
	To     sdk_types.AccAddress
	Amount sdk_types.Coins
}

func (am *AppModule) DefaultPaymentDistributionFunc(sdkctx sdk_types.Context, auctionPayment, totalBalance sdk_types.Coins) ([]Payment, error) {
	return []Payment{
		{
			To:     am.accountKeeper.GetModuleAddress(sdk_x_distribution_types.ModuleName),
			Amount: totalBalance,
		},
	}, nil
}

func (am AppModule) EndBlock(sdkctx sdk_types.Context, req tm_abci_types.RequestEndBlock) []tm_abci_types.ValidatorUpdate {
	logger := sdkctx.Logger().With("module", "x/mev", "method", "EndBlock", "height", strconv.Itoa(int(req.GetHeight())))
	logger.Debug("starting")
	defer func(begin time.Time) { logger.Debug("finished", "took", time.Since(begin)) }(time.Now())
	sdkctx = sdkctx.WithLogger(logger)

	paymentFunc := am.paymentFunc
	if paymentFunc == nil {
		paymentFunc = am.DefaultPaymentDistributionFunc
	}

	builderModuleAccountAddress := am.accountKeeper.GetModuleAddress("builder")
	if builderModuleAccountAddress == nil {
		logger.Error("payment distribution failed", "err", "builder module account address lookup failed")
		return nil
	}

	var auctionPayment sdk_types.Coins // TODO: this is best-effort, so maybe worth dropping altogether?
	if sc, ok := am.keeper.GetSegmentCommitmentByHeight(sdkctx, req.GetHeight()); ok {
		if payment, err := safeParseCoinNormalized(sc.PaymentPromise); err == nil {
			auctionPayment = sdk_types.Coins{payment}
		}
	}

	totalBalance := am.bankKeeper.SpendableCoins(sdkctx, builderModuleAccountAddress)

	logger.Info("calculating distribution",
		"builder_module_addr", builderModuleAccountAddress,
		"auction_payment", auctionPayment.String(),
		"total_balance", totalBalance.String(),
	)

	payments, err := paymentFunc(sdkctx, auctionPayment, totalBalance)
	if err != nil {
		logger.Error("payment distribution failed", "err", err)
		return nil
	}

	logger.Debug("calculated distribution", "payment_count", len(payments))

	for i, payment := range payments {
		err := am.bankKeeper.SendCoins(sdkctx, builderModuleAccountAddress, payment.To, payment.Amount)
		switch {
		case err == nil:
			logger.Debug(fmt.Sprintf("distribution payment %d/%d", i+1, len(payments)),
				"to", payment.To.String(),
				"amount", payment.Amount.String(),
				"result", "success",
			)
		case err != nil:
			logger.Error(fmt.Sprintf("distribution payment %d/%d", i+1, len(payments)),
				"to", payment.To.String(),
				"amount", payment.Amount.String(),
				"result", "failed",
				"err", err,
			)
		}
	}

	params := am.keeper.GetParams(sdkctx)
	minHeight := sdkctx.BlockHeight() - params.MaxEvidenceAgeNumBlocks
	am.keeper.DeleteOldSegmentCommitments(sdkctx, minHeight)
	am.keeper.DeleteOldProposerInfractions(sdkctx, minHeight)

	return nil
}
