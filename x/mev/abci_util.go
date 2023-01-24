package mev

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"sort"

	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	tm_types "github.com/cometbft/cometbft/types"
	sdk_crypto_types "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/exp/slices"
)

type SimulateFunc func(tx []byte) (sdk_types.GasInfo, *sdk_types.Result, error)

type PrefixFunc func(ctx context.Context, txs [][]byte) ([][]byte, error)

//
//
//

// Preference is an application-defined unique ID, and a corresponding
// validation function checking that preference over a set of transactions.
type Preference struct {
	ID           string
	ValidateFunc func(txs [][]byte) error
}

type Preferences []Preference

//

type preferences struct {
	slice []Preference
	index map[string]Preference
}

func newPreferences(prefs ...Preference) *preferences {
	ps := &preferences{
		slice: prefs,
		index: map[string]Preference{},
	}
	for _, p := range ps.slice {
		ps.index[p.ID] = p
	}
	return ps
}

func (ps preferences) getIDs() []string {
	ids := make([]string, len(ps.slice))
	for i, pref := range ps.slice {
		ids[i] = pref.ID
	}
	return ids
}

func (ps preferences) getPreference(id string) (Preference, bool) {
	p, ok := ps.index[id]
	return p, ok
}

//
//
//

// AuctionFunc takes a set of bids for an auction, and picks a single winning
// bid. To reject all bids, return ErrAuctionNoWinner.
type AuctionFunc func(ctx context.Context, bids []*AuctionBid) (*AuctionBid, error)

// ErrAuctionNoWinner indicates that an auction has no winning bid.
var ErrAuctionNoWinner = errors.New("auction: rejected all bids, no winner")

// DefaultAuctionFunc selects a winning bid by highest payment, breaking ties by
// highest transaction count.
func DefaultAuctionFunc(ctx context.Context, bids []*AuctionBid) (*AuctionBid, error) {
	if len(bids) <= 0 {
		return nil, ErrAuctionNoWinner
	}

	sort.SliceStable(bids, func(i, j int) bool {
		var (
			iPaymentAmount    = bids[i].Payment.Amount.Int64()
			jPaymentAmount    = bids[j].Payment.Amount.Int64()
			iTransactionCount = bids[i].TransactionCount
			jTransactionCount = bids[j].TransactionCount
		)
		switch {
		case iPaymentAmount > jPaymentAmount:
			return true
		case iPaymentAmount < jPaymentAmount:
			return false
		case iPaymentAmount == jPaymentAmount && iTransactionCount > jTransactionCount:
			return true
		case iPaymentAmount == jPaymentAmount && iTransactionCount < jTransactionCount:
			return false
		default:
			return false
		}
	})

	return bids[0], nil
}

// AuctionBid describes a bid for a specific block, made by a specific bidder.
// It's used by the AuctionFunc, which selects the winning bid for an auction.
//
// TODO: just a note that this is deliberately a separate type from BidRequest/BidResponse
type AuctionBid struct {
	// ChainID of the network holding the auction.
	ChainID string

	// Height of the block being auctioned.
	Height int64

	// PrefixTransactions are the set of transactions that will be at the top of
	// the block, directly before the transactions from the bid.
	PrefixTransactions [][]byte

	// PreferenceIDs sent to the builder by the proposer, which are the
	// application-defined rules that bids must satisfy.
	PreferenceIDs []string

	// Builder that produced the bid.
	Builder sdk_x_builder_types.Builder

	// TransactionCount is the number of transactions in the bid.
	TransactionCount int

	// Payment offered by the builder for the bid. The payment is guaranteed to
	// be a valid Coin, but Denom can be an empty string, and Amount can be
	// zero.
	Payment sdk_types.Coin
}

//
//
//

type Verifiable interface {
	VerifySignature(pubkey sdk_crypto_types.PubKey) bool
}

//
//
//

func cleanHTTPError(err error) error {
	if urlErr := (&url.Error{}); errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	return err
}

func safeParseCoinNormalized(coinStr string) (_ sdk_types.Coin, err error) {
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("parse coin (panic): %v", x)
		}
	}()

	coin, err := sdk_types.ParseCoinNormalized(coinStr)
	if err != nil {
		return sdk_types.Coin{}, fmt.Errorf("parse coin: %w", err)
	}

	return coin, nil
}

// TODO: move to types package
func validateBidResponse(req *sdk_x_builder_types.BidRequest, res *sdk_x_builder_types.BidResponse) error {
	if res.ChainID != req.ChainID {
		return fmt.Errorf("bid chain ID mismatch: want %q, have %q", req.ChainID, res.ChainID)
	}

	if res.Height != req.Height {
		return fmt.Errorf("bid height mismatch: want %q, have %q", req.Height, res.Height)
	}

	if !slices.Equal(req.PreferenceIDs, res.PreferenceIDs) {
		return fmt.Errorf("bid preferences mismatch: want %v, have %v", req.PreferenceIDs, res.PreferenceIDs)
	}

	if want, have := sdk_x_builder_types.HashByteSlices(req.PrefixTransactions...), res.PrefixHash; !bytes.Equal(want, have) {
		return fmt.Errorf("bid prefix hash mismatch: want %X, have %X", want, have)
	}

	if res.PaymentPromise == "" {
		return fmt.Errorf("bid payment promise empty")
	}

	if res.SegmentLength < 0 {
		return fmt.Errorf("bid segment length invalid: %d", res.SegmentLength)
	}

	if want, have := sha256.Size, len(res.SegmentHash); want != have {
		return fmt.Errorf("bid segment hash size mismatch: want %d, have %d", want, have)
	}

	return nil
}

//
//
//

type validateCommitmentConfig struct {
	BlockTxs [][]byte

	Required     bool         // if true, a missing segment commitment is returned as an error
	SignerAddr   string       // optional, if provided, it's compared with MsgCommitSegment's signer
	SimulateFunc SimulateFunc // optional, if provided, it's called with the commitment's tx bytes
}

func (am *AppModule) validateCommitment(cfg validateCommitmentConfig) error {
	var (
		commitMsg *sdk_x_builder_types.MsgCommitSegment
		commitTx  []byte
	)
	for i, txBytes := range cfg.BlockTxs {
		tx, err := am.txDecoder(txBytes)
		if err != nil {
			continue
		}

		msgs := tx.GetMsgs()
		if len(msgs) != 1 {
			continue
		}

		m, ok := msgs[0].(*sdk_x_builder_types.MsgCommitSegment)
		if !ok {
			continue
		}

		if want, have := m.Commitment.SegmentOffset+m.Commitment.SegmentLength, int32(i); want != have {
			return fmt.Errorf("invalid position in block: want %d, have %d", want, have)
		}

		signers := m.GetSigners()
		if want, have := 1, len(signers); want != have {
			return fmt.Errorf("segment commitment signer count: want %d, have %d", want, have)
		}

		if cfg.SignerAddr != "" {
			if want, have := cfg.SignerAddr, signers[0].String(); want != have {
				return fmt.Errorf("segment commitment signer: want %q, have %q", want, have)
			}
		}

		commitMsg = m
		commitTx = txBytes
		break
	}

	if commitMsg == nil {
		switch {
		case cfg.Required:
			return fmt.Errorf("no segment commitment found in block")
		default:
			return nil
		}
	}

	if cfg.SimulateFunc != nil {
		if _, _, err := cfg.SimulateFunc(commitTx); err != nil {
			return fmt.Errorf("simulation failed: %w", err)
		}
	}

	if err := commitMsg.Commitment.VerifyBlockHashes(cfg.BlockTxs); err != nil {
		return fmt.Errorf("verify block hashes: %w", err)
	}

	for _, id := range commitMsg.Commitment.PreferenceIds {
		p, ok := am.preferences.getPreference(id)
		if !ok {
			return fmt.Errorf("preference %q is unknown", id)
		}
		if err := p.ValidateFunc(cfg.BlockTxs); err != nil {
			return fmt.Errorf("preference %q validation error: %w", id, err)
		}
	}

	return nil
}

//
//
//

func decodeTransactionExtra(ctx context.Context, decoder sdk_types.TxDecoder, txb []byte) (_ sdk_types.Tx, size, gas int64, _ error) {
	sdktx, err := decoder(txb)
	if err != nil {
		return nil, 0, 0, err
	}

	size = tm_types.ComputeProtoSizeForTxs([]tm_types.Tx{txb})

	if g, ok := sdktx.(interface{ GetGas() uint64 }); ok {
		gas = int64(g.GetGas())
	}

	return sdktx, size, gas, nil
}
