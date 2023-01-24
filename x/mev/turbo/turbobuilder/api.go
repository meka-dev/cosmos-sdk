package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	sdk_x_builder "cosmossdk.io/x/mev"
	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	tm_types "github.com/cometbft/cometbft/types"
	sdk_client_tx "github.com/cosmos/cosmos-sdk/client/tx"
	sdk_crypto_types "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	sdk_types_tx_signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	sdk_x_auth_signing "github.com/cosmos/cosmos-sdk/x/auth/signing"
	sdk_x_auth_types "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type API struct {
	key   *sdk_x_builder_types.Key
	node  APINode
	store Store
}

type APINode interface {
	Proposer(ctx context.Context, height int64, proposerAddr string) (*sdk_x_builder_types.QueryProposerResponse, error)
	AccountAtHeight(ctx context.Context, height int64, addr string) (*sdk_x_auth_types.BaseAccount, error)
}

func NewAPI(ctx context.Context, key *sdk_x_builder_types.Key, n APINode, s Store) (*API, error) {
	return &API{
		key:   key,
		node:  n,
		store: s,
	}, nil
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v0/bid", a.handleV0Bid)
	mux.HandleFunc("/v0/commit", a.handleV0Commit)
	mux.ServeHTTP(w, r)
}

func (a *API) handleV0Bid(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req sdk_x_builder_types.BidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(ctx, w, http.StatusBadRequest, fmt.Errorf("parse bid request: %w", err))
		return
	}

	log.Printf("bid request: ProposerAddress=%s ChainID=%s Height=%d", req.ProposerAddress, req.ChainID, req.Height)

	// Builders should validate the incoming bid, including e.g. checking the
	// chain ID and height, as well as verifying the signature. The builder
	// should query a trusted authority (full node, light client, etc.) to get
	// the valid proposers for the bid height, verify that the bid's signature
	// matches one of them.
	//
	// This process should be followed for all builder API endpoints.

	if err := a.validateRequest(ctx, req.Height, req.ProposerAddress, &req); err != nil {
		respondError(ctx, w, http.StatusBadGateway, fmt.Errorf("validate bid request: %w", err))
		return
	}

	// Once a bid request is verified, the builder makes a bid for the auction.
	//
	// Once placed, a bid is immutable, and the same bid must always be returned
	// to all requests for the corresponding auction. Therefore, builders should
	// persist bids, and only create a new bid if one does not already exist.

	bid, err := a.store.SelectBid(ctx, req.ProposerAddress, req.ChainID, req.Height) // TODO: transactional semantics here with the store
	switch {
	case errors.Is(err, ErrBidNotFound):
		bid, err = a.computeBid(ctx, &req)
		if err != nil {
			respondError(ctx, w, http.StatusInternalServerError, fmt.Errorf("create bid: %w", err))
		}
		if err := a.store.InsertBid(ctx, bid); err != nil {
			respondError(ctx, w, http.StatusInternalServerError, fmt.Errorf("save bid: %w", err))
			return
		}

	case err != nil:
		respondError(ctx, w, http.StatusInternalServerError, fmt.Errorf("lookup bid: %w", err))
		return
	}

	// The builder then signs and returns the bid response.

	res := &sdk_x_builder_types.BidResponse{
		ProposerAddress: bid.ProposerAddress,
		ChainID:         bid.ChainID,
		Height:          req.Height,
		PrefixHash:      bid.PrefixHash,
		PreferenceIDs:   bid.Preferences,
		SegmentLength:   bid.SegmentLength,
		SegmentBytes:    bid.SegmentBytes,
		SegmentGas:      bid.SegmentGas,
		SegmentHash:     bid.SegmentHash,
		PaymentPromise:  bid.PaymentPromise,
	}
	if err := res.SignWith(a.key); err != nil {
		respondError(ctx, w, http.StatusInternalServerError, fmt.Errorf("sign bid response: %w", err))
		return
	}

	respondJSON(ctx, w, http.StatusOK, res)
}

func (a *API) handleV0Commit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req sdk_x_builder_types.CommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(ctx, w, http.StatusBadRequest, fmt.Errorf("decode commit request: %w", err))
		return
	}

	if err := a.validateRequest(ctx, req.Height, req.ProposerAddress, &req); err != nil {
		respondError(ctx, w, http.StatusBadGateway, fmt.Errorf("validate commit request: %w", err))
		return
	}

	bid, err := a.store.SelectBid(ctx, req.ProposerAddress, req.ChainID, req.Height)
	if err != nil {
		respondError(ctx, w, http.StatusBadRequest, fmt.Errorf("commit to unknown bid"))
		return
	}

	segmentCommitment := sdk_x_builder_types.SegmentCommitment{
		ProposerAddress:   req.ProposerAddress,
		BuilderAddress:    req.BuilderAddress,
		ChainId:           req.ChainID,
		Height:            req.Height,
		PreferenceIds:     req.PreferenceIDs,
		PrefixOffset:      req.PrefixOffset,
		PrefixLength:      req.PrefixLength,
		PrefixHash:        req.PrefixHash,
		SegmentOffset:     req.SegmentOffset,
		SegmentLength:     req.SegmentLength,
		SegmentBytes:      int32(req.SegmentBytes), // TODO: change to int64 or uint64
		SegmentGas:        int32(req.SegmentGas),   // TODO: change to int64 or uint64
		SegmentHash:       req.SegmentHash,
		PaymentPromise:    req.PaymentPromise,
		ProposerSignature: req.Signature,
	}

	// TODO: Verify that all given segment commitment fields correspond to what we were shown before.

	builderSignature, err := a.key.Sign(segmentCommitment.GetSignBytes())
	if err != nil {
		respondError(ctx, w, http.StatusInternalServerError, fmt.Errorf("sign segment commitment: %w", err))
		return
	}

	segmentCommitment.BuilderSignature = builderSignature

	info, err := a.node.AccountAtHeight(ctx, req.Height-1, a.key.Address)
	if err != nil {
		respondError(ctx, w, http.StatusInternalServerError, fmt.Errorf("get builder account sequence: %w", err))
		return
	}

	seq := info.GetSequence()

	segmentCommitmentTx, err := buildTx(ctx, a.key, req.ChainID, info.GetAccountNumber(), seq,
		&sdk_x_builder_types.MsgCommitSegment{
			BuilderAddress: a.key.Address,
			Commitment:     segmentCommitment,
		},
	)
	if err != nil {
		respondError(ctx, w, http.StatusInternalServerError, fmt.Errorf("build commitment transaction: %w", err))
		return
	}

	bid.SegmentCommitment = &segmentCommitment
	bid.SegmentCommitmentTransaction = segmentCommitmentTx
	bid.State = BidStateWon

	if err := a.store.UpdateBid(ctx, bid); err != nil {
		respondError(ctx, w, http.StatusBadRequest, fmt.Errorf("update bid in store: %w", err))
		return
	}

	res := &sdk_x_builder_types.CommitResponse{
		ChainID:                      req.ChainID,
		Height:                       req.Height,
		SegmentTransactions:          bid.SegmentTransactions,
		SegmentCommitmentTransaction: bid.SegmentCommitmentTransaction,
	}

	if err := res.SignWith(a.key); err != nil {
		respondError(ctx, w, http.StatusInternalServerError, fmt.Errorf("sign commit response: %w", err))
		return
	}

	respondJSON(ctx, w, http.StatusOK, res)
}

func (a *API) computeBid(ctx context.Context, req *sdk_x_builder_types.BidRequest) (*Bid, error) {
	var initialTxs [][]byte
	{
		// Here is where the builder would fetch transactions it received from
		// e.g. searchers targeting the auction represented by the bid request.
	}

	var (
		segmentTxs   [][]byte
		segmentBytes int64
		segmentGas   int64
	)
	for i, txBytes := range initialTxs {
		sdkTx, err := encodingConfig.TxConfig.TxDecoder()(txBytes)
		if err != nil {
			log.Printf("decode bid segment tx %d/%d: %v", i+1, len(initialTxs), err)
			continue // TODO
		}

		txSizeBytes := tm_types.ComputeProtoSizeForTxs([]tm_types.Tx{txBytes})
		if req.MaxBytes > 0 && segmentBytes+txSizeBytes > req.MaxBytes {
			log.Printf("size (%d) + segment tx %d/%d size (%d) would exceed max (%d)", segmentBytes, i+1, len(initialTxs), txSizeBytes, req.MaxBytes)
			break
		}

		var txGasUsed int64
		if g, ok := sdkTx.(interface{ GetGas() uint64 }); ok {
			txGasUsed = int64(g.GetGas())
		}
		if req.MaxGas > 0 && segmentGas+txGasUsed > req.MaxGas {
			log.Printf("gas (%d) + segment tx %d/%d gas (%d) would exceed max (%d)", segmentGas, i+1, len(initialTxs), txGasUsed, req.MaxGas)
			break
		}

		segmentTxs = append(segmentTxs, txBytes)
		segmentBytes += txSizeBytes
		segmentGas += txGasUsed
	}

	// TODO: need to account for segment commitment tx, but maybe not here

	return &Bid{
		ProposerAddress:     req.ProposerAddress,
		ChainID:             req.ChainID,
		Height:              req.Height,
		Preferences:         req.PreferenceIDs,
		PrefixHash:          hashOf(req.PrefixTransactions...),
		PrefixLength:        len(req.PrefixTransactions),
		SegmentHash:         hashOf(segmentTxs...),
		SegmentLength:       len(segmentTxs),
		SegmentBytes:        segmentBytes,
		SegmentGas:          segmentGas,
		PaymentPromise:      "250" + req.PaymentDenom,
		SegmentTransactions: segmentTxs,
		State:               BidStatePending,
	}, nil
}

var debugMemoCounter atomic.Uint64

func buildTx(ctx context.Context, key *sdk_x_builder_types.Key, chainID string, accNum, seq uint64, msgs ...sdk_types.Msg) ([]byte, error) {
	// https://docs.cosmos.network/main/run-node/txs#programmatically-with-go

	txBuilder := encodingConfig.TxConfig.NewTxBuilder()

	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, fmt.Errorf("set transaction messages: %w", err)
	}

	txBuilder.SetGasLimit(100_000) // TODO: this isn't right

	txBuilder.SetFeeAmount(sdk_types.Coins{
		{
			Amount: sdk_types.NewInt(5), // TODO: need to simulate to get proper value?
			Denom:  "stake",             // TODO: get from original bid request
		},
	})

	txBuilder.SetMemo(fmt.Sprintf("tx %d", debugMemoCounter.Add(1)))

	// txBuilder.SetTimeoutHeight()

	preSignature := sdk_types_tx_signing.SignatureV2{
		PubKey: key.PubKey,
		Data: &sdk_types_tx_signing.SingleSignatureData{
			SignMode:  encodingConfig.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: seq,
	}

	if err := txBuilder.SetSignatures(preSignature); err != nil {
		return nil, fmt.Errorf("set signatures (initial): %w", err)
	}

	signerData := sdk_x_auth_signing.SignerData{
		ChainID:       chainID,
		AccountNumber: accNum,
		Sequence:      seq,
	}

	postSignature, err := sdk_client_tx.SignWithPrivKey(
		ctx,
		encodingConfig.TxConfig.SignModeHandler().DefaultMode(),
		signerData,
		txBuilder,
		key.PrivKey,
		encodingConfig.TxConfig,
		seq,
	)
	if err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	if err := txBuilder.SetSignatures(postSignature); err != nil {
		return nil, fmt.Errorf("set signatures (signed): %w", err)
	}

	txBytes, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("encode signed transaction: %w", err)
	}

	return txBytes, nil
}

func (a *API) validateRequest(ctx context.Context, height int64, proposerAddr string, req sdk_x_builder.Verifiable) error {
	p, err := a.node.Proposer(ctx, height-1, proposerAddr)
	if err != nil {
		return fmt.Errorf("get proposer for auction: %w", err)
	}

	if p.Validator == nil {
		return fmt.Errorf("proposer %s not in staking validator set at height %d", proposerAddr, height)
	}

	if !p.Validator.IsBonded() {
		return fmt.Errorf("proposer %s not bonded at height %d", proposerAddr, height)
	}

	if p.Validator.IsJailed() {
		return fmt.Errorf("proposer %s jailed at height %d", proposerAddr, height)
	}

	if !req.VerifySignature(p.Proposer.Pubkey.GetCachedValue().(sdk_crypto_types.PubKey)) {
		return fmt.Errorf("bad signature")
	}

	return nil
}

func respondJSON(ctx context.Context, w http.ResponseWriter, code int, res any) {
	data, err := json.Marshal(res)
	if err != nil {
		respondError(ctx, w, http.StatusInternalServerError, fmt.Errorf("marshal %d response: %w", code, err))
		return
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func respondError(ctx context.Context, w http.ResponseWriter, code int, err error) {
	data, _ := json.Marshal(map[string]any{"error": err.Error()})
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}
