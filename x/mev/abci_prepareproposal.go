package mev

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	tm_abci_types "github.com/cometbft/cometbft/abci/types"
	tm_libs_log "github.com/cometbft/cometbft/libs/log"
	sdk_crypto_types "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
)

func (am *AppModule) PrepareProposal(sdkctx sdk_types.Context, req tm_abci_types.RequestPrepareProposal) tm_abci_types.ResponsePrepareProposal {
	height := req.GetHeight()

	defer func() { // cleanup
		for h := range am.memoizedAuctions {
			if h < height-1 {
				delete(am.memoizedAuctions, h)
			}
		}
	}()

	// Technically it's incorrect to memoize auctions by height alone: auctions
	// are uniquely identified by the 3-tuple of proposer, chain ID, and height.
	// But proposer and chain ID are implicit and well-defined here.

	res, ok := am.memoizedAuctions[height]
	if !ok {
		res = am.runAuction(sdkctx, req)
		am.memoizedAuctions[height] = res
	}

	return res
}

//
//
//

func (am *AppModule) runAuction(sdkctx sdk_types.Context, req tm_abci_types.RequestPrepareProposal) tm_abci_types.ResponsePrepareProposal {
	// Initial setup, including the regular context and a logger.
	logger := sdkctx.Logger().With("module", "x/mev", "method", "PrepareProposal", "height", strconv.Itoa(int(req.GetHeight())))
	logger.Debug("starting")
	defer func(begin time.Time) { logger.Debug("finished", "took", time.Since(begin)) }(time.Now())
	sdkctx = sdkctx.WithLogger(logger)
	ctx := sdkctx.Context()

	// Log some details about the request.
	logger.Debug("metadata",
		"sdkctx.ChainID", sdkctx.ChainID(),
		"sdkctx.BlockHeight", sdkctx.BlockHeight(),
		"req.max_tx_bytes", req.MaxTxBytes,
		"req.txs_count", len(req.Txs),
		"req.time", req.Time,
		"req.proposer_address", sdk_types.AccAddress(req.ProposerAddress), // public key of the proposer
	)

	// Define a timeout which is << the block rate for the chain. This is a
	// (weak) way of asserting a time limit on the entire function, with the
	// goal of preventing e.g. slashing.
	prepareTimeout := 3 * time.Second // TODO: this needs to be << the config.toml [consensus] prepare timeout, with a minimum of ~1s
	ctx, cancel := context.WithTimeout(ctx, prepareTimeout)
	defer cancel()

	sdkctx = sdkctx.WithContext(ctx)

	// decodeTx := am.txEncodingConfig.TxDecoder() // TODO: Put this in a field in AppModule
	decodeTx := am.txDecoder

	// Create a default response, based on the request, which we fall back to in
	// case of any problem or error.
	defaultResponse := tm_abci_types.ResponsePrepareProposal{
		Txs: filterSegmentCommitmentTxs(decodeTx, req.Txs),
	}

	// The first call to PrepareProposal on startup can have an empty chain ID.
	if sdkctx.ChainID() == "" {
		logger.Error("PrepareProposal called with empty chain ID -- this is a bug in the SDK")
		return defaultResponse
	}

	// Check if this proposer already registered.
	proposer, ok := am.keeper.GetProposer(sdkctx, am.proposerKey.Address)
	if !ok {
		logger.Info("proposer isn't registered")
		return defaultResponse
	}

	logger.Debug("proposer addresses",
		"request_pubkey_addr", sdk_types.AccAddress(req.ProposerAddress),
		"module_proposer_key_addr", am.proposerKey.Address,
		"keeper_proposer_addr", proposer.Address,
		"keeper_proposer_operator_addr", proposer.OperatorAddress,
		"keeper_proposer_pubkey_addr", sdk_types.AccAddress(proposer.Pubkey.GetCachedValue().(sdk_crypto_types.PubKey).Address()),
	)

	// Fetch registered and allowed builders for the auction.
	builders := am.keeper.GetAuctionBuilders(sdkctx)
	if len(builders) == 0 {
		logger.Info("not soliciting bids, no allowed builders")
		return defaultResponse
	}

	// The bid request we send to builders includes the prefix region, which is
	// a set of transactions that will be top-of-block and come before whatever
	// transactions are in the bid. They can be empty.
	prefixTransactions, err := am.prefixFunc(ctx, req.Txs)
	if err != nil {
		logger.Error("error getting prefix txs", "err", err)
		return defaultResponse
	}

	// MsgCommitSegment is only allowed to exist in the commit transaction that
	// we create, so filter out any transactions containing that message in both
	// the prefix region and the mempool transactions from the request.
	prefixTransactions = filterSegmentCommitmentTxs(decodeTx, prefixTransactions)
	mempoolTransactions := filterSegmentCommitmentTxs(decodeTx, req.Txs)

	// We only accept bids in the base denom for the chain.
	paymentDenom, err := sdk_types.GetBaseDenom()
	if err != nil {
		logger.Debug("error getting base denom", "err", err, "default", sdk_types.DefaultBondDenom)
		paymentDenom = sdk_types.DefaultBondDenom
	}

	// Create, and sign, the bid request we will send to builders.
	bidRequest := &sdk_x_builder_types.BidRequest{
		ProposerAddress:    am.proposerKey.Address,
		ChainID:            sdkctx.ChainID(),
		Height:             req.Height,
		PaymentDenom:       paymentDenom,
		PreferenceIDs:      am.preferences.getIDs(),
		PrefixTransactions: prefixTransactions,
		MaxBytes:           req.MaxTxBytes,                        // TODO: Subtract prefix space / gas and reserve some space / gas for mempool txs.
		MaxGas:             sdkctx.ConsensusParams().Block.MaxGas, // TODO: do we need to enforce this here?
	}
	if err := bidRequest.SignWith(am.proposerKey); err != nil {
		logger.Error("error signing bid request", "err", err)
		return defaultResponse
	}

	logger.Debug("signed bid request",
		"signature", fmt.Sprintf("%x", bidRequest.Signature),
		"builder_count", len(builders),
	)

	// Scatter that request to the selected builders.
	acceptedBids, err := am.getBids(sdkctx, builders, bidRequest, am.httpClient, logger)
	if err != nil {
		logger.Error("error getting bids from builders", "err", err)
		return defaultResponse
	}

	// Choose a winning bid from the valid responses.
	var winningBid *bidTuple
	{
		// Extract auction bids from the accepted bid tuples.
		// TODO: probably a better way to manage this mapping
		auctionBids := make([]*AuctionBid, len(acceptedBids))
		for i := range acceptedBids {
			auctionBids[i] = acceptedBids[i].AuctionBid
		}

		// Pass those auction bids to the auction func.
		winningAuctionBid, err := am.auctionFunc(ctx, auctionBids)
		if err != nil {
			logger.Error("error choosing winning bid", "err", err)
			return defaultResponse
		}

		// Map the winning auction bid back to its bid tuple.
		for _, bid := range acceptedBids {
			if bid.AuctionBid == winningAuctionBid {
				winningBid = bid
				break
			}
		}

		if winningBid == nil {
			logger.Error("mismatch selecting winning bid: programmer error")
			return defaultResponse
		}
	}

	logger.Debug("selected winning bid",
		"builder_moniker", winningBid.Builder.Moniker,
		"builder_address", winningBid.Builder.Address,
		"payment", winningBid.AuctionBid.Payment.String(),
	)

	// The winning bid must be validated. That includes things like: no segment
	// commitment transaction, correct bytes/gas, can make payment, etc.
	//
	// If the winning bid is invalid, we just fall back to the default response.
	// It would be possible to select a "runner up" bid, if the auction func
	// returned "ranked choice" bids rather than just a single winner.
	{
		// TODO: validate winning bid
		logger.Debug("winning bid metadata",
			"payment_promise", winningBid.Response.PaymentPromise,
			"segment_length", winningBid.Response.SegmentLength,
			"segment_bytes", winningBid.Response.SegmentBytes,
		)
	}

	// Send a commitment to the winning builder for their bid. They'll respond
	// with the bid transactions, and a segment commitment transaction, which
	// goes into the block.
	if err := winningBid.makeCommitment(ctx, am.proposerKey, am.httpClient, logger); err != nil {
		logger.Error("error making commitment to winning builder", "err", err)
		return defaultResponse
	}

	logger.Debug("made commitment to winning bid",
		"builder_moniker", winningBid.Builder.Moniker,
		"builder_address", winningBid.Builder.Address,
	)

	// TODO: From this point on, make sure to include a tx with evidence against
	// the builder for any errors that are their fault.

	// Now we should be able to get the segment data from the winning bid.
	segmentTransactions, segmentCommitmentTransaction, err := winningBid.getSegment(ctx)
	if err != nil {
		logger.Error("error fetching segment for winning bid", "err", err)
		return defaultResponse
	}

	logger.Debug("got bid segment",
		"tx_count", len(segmentTransactions),
		"commitment_tx_len", len(segmentCommitmentTransaction),
	)

	// Now we can build the block. The returned transactions satisfy the
	// size/gas limits, but deeper validation of e.g. commitment rules are
	// performed later.
	blockTransactions, err := getBlockTransactions(
		sdkctx,
		decodeTx,
		bidRequest.MaxBytes,
		bidRequest.MaxGas,
		prefixTransactions,
		segmentTransactions,
		segmentCommitmentTransaction,
		mempoolTransactions,
	)
	if err != nil {
		logger.Error("error computing block transactions", "err", err)
		return defaultResponse
	}

	logger.Debug("computed block transactions",
		"n", len(blockTransactions),
	)

	// Now that a tentative block is built, we can validate that it satisfies
	// all of the more semantic requirements, like preferences, which are
	// described by the segment commitment metadata.
	if err := am.validateCommitment(validateCommitmentConfig{
		BlockTxs:     blockTransactions,
		Required:     true,
		SignerAddr:   winningBid.Builder.Address,
		SimulateFunc: am.simulatePrepareFunc,
	}); err != nil {
		logger.Error("error validating proposal txs", "err", err)
		return defaultResponse
	}

	logger.Debug("validated block transactions",
		"n", len(blockTransactions),
	)

	// And we're done.
	return tm_abci_types.ResponsePrepareProposal{
		Txs: blockTransactions,
	}
}

// filterSegmentCommitmentTxs removes any txs containing MsgCommitSegment messages
// from the given list of transactions.
func filterSegmentCommitmentTxs(decode sdk_types.TxDecoder, txs [][]byte) [][]byte {
	var pass [][]byte
	for _, txBytes := range txs {
		sdkTx, err := decode(txBytes)
		if err != nil {
			continue // TODO: should we default-allow instead?
		}

		var found bool
		for _, msg := range sdkTx.GetMsgs() {
			if _, found = msg.(*sdk_x_builder_types.MsgCommitSegment); found {
				break
			}
		}
		if found {
			continue
		}

		pass = append(pass, txBytes)
	}
	return pass
}

//
//
//

// bidTuple is how PrepareProposal tracks a bid from a specific builder. It
// starts in a raw form and is modified over time, as the bid is verified,
// evaluated, and maybe committed-to.
type bidTuple struct {
	Builder    *sdk_x_builder_types.Builder
	Request    *sdk_x_builder_types.BidRequest
	Response   *sdk_x_builder_types.BidResponse
	AuctionBid *AuctionBid
	Commitment *sdk_x_builder_types.CommitResponse
	Error      error
}

func (bt *bidTuple) postprocess(sdkctx sdk_types.Context, bankKeeper sdk_x_builder_types.BankKeeper) error {
	if err := validateBidResponse(bt.Request, bt.Response); err != nil {
		return fmt.Errorf("validate bid response: %w", err)
	}

	payment, err := safeParseCoinNormalized(bt.Response.PaymentPromise)
	if err != nil {
		return fmt.Errorf("parse payment promise: %w", err)
	}

	if want, have := bt.Request.PaymentDenom, payment.Denom; want != have {
		return fmt.Errorf("payment denom: want %q, have %q", want, have)
	}

	builderAddress, err := sdk_types.AccAddressFromBech32(bt.Builder.Address)
	if err != nil {
		return fmt.Errorf("parse builder address: %w", err)
	}

	balance := bankKeeper.SpendableCoins(sdkctx, builderAddress)
	if payment.Amount.GT(balance.AmountOfNoDenomValidation(payment.Denom)) {
		return fmt.Errorf("insufficient builder account balance")
	}

	bt.AuctionBid = &AuctionBid{
		ChainID:            bt.Request.ChainID,
		Height:             bt.Request.Height,
		PrefixTransactions: bt.Request.PrefixTransactions,
		PreferenceIDs:      bt.Request.PreferenceIDs,
		Builder:            *bt.Builder,
		TransactionCount:   bt.Response.SegmentLength,
		Payment:            payment,
	}

	return nil
}

func (bt *bidTuple) makeCommitment(
	ctx context.Context,
	proposerKey *sdk_x_builder_types.ProposerKey,
	httpClient HTTPClient,
	logger tm_libs_log.Logger,
) error {
	req := &sdk_x_builder_types.CommitRequest{
		ProposerAddress: proposerKey.Address,
		BuilderAddress:  bt.Builder.Address,
		ChainID:         bt.Response.ChainID,
		Height:          bt.Response.Height,
		PreferenceIDs:   bt.Response.PreferenceIDs,
		PrefixOffset:    0,
		PrefixLength:    int32(len(bt.Request.PrefixTransactions)),
		PrefixHash:      bt.Response.PrefixHash,
		SegmentOffset:   int32(len(bt.Request.PrefixTransactions)),
		SegmentLength:   int32(bt.Response.SegmentLength),
		SegmentBytes:    bt.Response.SegmentBytes,
		SegmentGas:      bt.Response.SegmentGas,
		SegmentHash:     bt.Response.SegmentHash,
		PaymentPromise:  bt.Response.PaymentPromise,
	}

	if err := req.SignWith(proposerKey); err != nil {
		return fmt.Errorf("error signing proposer commit request: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling commit request: %w", err)
	}

	var res sdk_x_builder_types.CommitResponse
	if err := sendBuilderRequest(ctx, httpClient, bt.Builder, "commit", body, &res, logger); err != nil {
		return fmt.Errorf("send commitment: %w", err)
	}

	bt.Commitment = &res

	return nil
}

func (bt *bidTuple) getSegment(ctx context.Context) ([][]byte, []byte, error) {
	if len(bt.Commitment.SegmentCommitmentTransaction) <= 0 {
		return nil, nil, fmt.Errorf("empty segment commitment transaction in bid")
	}

	return bt.Commitment.SegmentTransactions, bt.Commitment.SegmentCommitmentTransaction, nil
}

//
//
//

func (am *AppModule) getBids(sdkctx sdk_types.Context, builders []sdk_x_builder_types.Builder, req *sdk_x_builder_types.BidRequest, httpClient HTTPClient, logger tm_libs_log.Logger) ([]*bidTuple, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode bid request: %w", err)
	}

	bidc := make(chan *bidTuple, len(builders))
	for i := range builders {
		go func(b *sdk_x_builder_types.Builder) {
			var res sdk_x_builder_types.BidResponse
			if err := sendBuilderRequest(sdkctx, httpClient, b, "bid", body, &res, logger); err != nil {
				bidc <- &bidTuple{Builder: b, Request: req, Response: nil, Error: err}
			} else {
				bidc <- &bidTuple{Builder: b, Request: req, Response: &res, Error: nil}
			}
		}(&builders[i])
	}

	var bids []*bidTuple
	for i := 0; i < cap(bidc); i++ {
		bid := <-bidc
		if bid.Error != nil {
			logger.Error("bid failed",
				"err", bid.Error,
				"builder_moniker", bid.Builder.Moniker,
				"builder_address", bid.Builder.Address,
			)
			continue
		}

		if err := bid.postprocess(sdkctx, am.bankKeeper); err != nil {
			logger.Error("bid invalid",
				"err", err,
				"builder_moniker", bid.Builder.Moniker,
				"builder_address", bid.Builder.Address,
			)
			continue
		}

		logger.Debug("bid accepted",
			"builder_moniker", bid.Builder.Moniker,
			"builder_address", bid.Builder.Address,
			"payment_promise", bid.Response.PaymentPromise,
		)
		bids = append(bids, bid)
	}

	if len(bids) <= 0 {
		return nil, fmt.Errorf("no successful bids")
	}

	return bids, nil
}

func getBlockTransactions(
	sdkctx sdk_types.Context,
	decode sdk_types.TxDecoder,
	maxSize, maxGas int64,
	prefixTransactions [][]byte,
	segmentTransactions [][]byte,
	segmentCommitmentTransaction []byte,
	mempoolTransactions [][]byte,
) ([][]byte, error) {
	type txhash [sha256.Size]byte

	index := map[txhash]struct{}{}

	observe := func(txb []byte) bool {
		h := sha256.Sum256(txb) // hash the tx
		_, seen := index[h]     // have we already seen it?
		index[h] = struct{}{}   // well we have now
		return seen             //
	}

	var (
		blockTxs  [][]byte
		blockSize int64
		blockGas  int64
	)

	var (
		errTooManyBytes = fmt.Errorf("transaction would exceed block size limit (%d)", maxSize)
		errTooMuchGas   = fmt.Errorf("transaction would exceed block gas limit (%d)", maxGas)
	)

	include := func(txb []byte) error {
		_, size, gas, err := decodeTransactionExtra(sdkctx, decode, txb)
		if err != nil {
			return fmt.Errorf("decode transaction: %w", err)
		}
		if maxSize > 0 && blockSize+size > maxSize {
			return errTooManyBytes
		}
		if maxGas > 0 && blockGas+gas > maxGas {
			return errTooMuchGas
		}
		blockTxs = append(blockTxs, txb)
		blockSize += size
		blockGas += gas
		return nil
	}

	for _, txb := range prefixTransactions {
		observe(txb)

		if err := include(txb); err != nil {
			return nil, fmt.Errorf("prefix: %w", err)
		}
	}

	sdkctx.Logger().Debug("filled prefix txs", "block_size", blockSize, "block_gas", blockGas)

	for _, txb := range append(segmentTransactions, segmentCommitmentTransaction) {
		observe(txb) // should we error on duplicates here?

		if err := include(txb); err != nil {
			return nil, fmt.Errorf("segment: %w", err)
		}
	}

	sdkctx.Logger().Debug("filled segment txs", "block_size", blockSize, "block_gas", blockGas)

	for _, txb := range mempoolTransactions {
		if observe(txb) {
			continue // don't include transactions we already have
		}

		if err := include(txb); errors.Is(err, errTooManyBytes) || errors.Is(err, errTooMuchGas) {
			break // we've reached the block limits
		} else if err != nil {
			return nil, fmt.Errorf("mempool: %w", err)
		}
	}

	sdkctx.Logger().Debug("filled mempool txs", "block_size", blockSize, "block_gas", blockGas)

	// TODO: do we need to somehow replace the mempool transactions we didn't take?

	return blockTxs, nil
}

func sendBuilderRequest(
	ctx context.Context,
	httpClient HTTPClient,
	builder *sdk_x_builder_types.Builder,
	path string,
	body []byte,
	res Verifiable,
	logger tm_libs_log.Logger,
) error {
	u, err := url.Parse(builder.BuilderApiUrl)
	if err != nil {
		return fmt.Errorf("builder API URL: %w", err)
	}

	newpath, err := url.JoinPath(u.Path, path)
	if err != nil {
		return fmt.Errorf("builder API URL: %w", err)
	}

	u.Path = newpath

	httpReq, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpRes, err := httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute request: %w", cleanHTTPError(err))
	}

	bodyReader := io.LimitReader(httpRes.Body, 100*1024*1024) // TODO: make limit const
	defer func() {
		io.Copy(io.Discard, bodyReader)
		httpRes.Body.Close()
	}()

	if httpRes.StatusCode != http.StatusOK {
		// TODO: read response body, log at debug level
		return fmt.Errorf("builder response code %d (%s)", httpRes.StatusCode, http.StatusText(httpRes.StatusCode))
	}

	if err := json.NewDecoder(bodyReader).Decode(res); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	pubKey := builder.Pubkey.GetCachedValue().(sdk_crypto_types.PubKey)
	if !res.VerifySignature(pubKey) {
		return fmt.Errorf("invalid response signaturetgit ")
	}

	logger.Debug("builder response signature OK",
		"builder_moniker", builder.Moniker,
		"builder_address", builder.Address,
		"response_type", fmt.Sprintf("%T", res),
	)

	return nil
}
