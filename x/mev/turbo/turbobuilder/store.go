package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	sdk_x_builder_types "cosmossdk.io/x/mev/types"
)

// Store is an interface that defines the methods for storing bids.
type Store interface {
	UpdateBid(ctx context.Context, bid *Bid) error
	InsertBid(ctx context.Context, bid *Bid) error
	SelectBid(ctx context.Context, proposerAddr, chainID string, height int64) (*Bid, error)
	ListBids(ctx context.Context) ([]*Bid, error)
	DeleteBid(ctx context.Context, proposerAddr, chainID string, height int64) error
}

// Bid is a struct that represents a bid.
type Bid struct {
	ProposerAddress string
	ChainID         string
	Height          int64

	Preferences         []string
	PrefixHash          []byte
	PrefixLength        int
	SegmentHash         []byte
	SegmentLength       int
	SegmentBytes        int64
	SegmentGas          int64
	SegmentTransactions [][]byte
	PaymentPromise      string

	SegmentCommitment            *sdk_x_builder_types.SegmentCommitment
	SegmentCommitmentTransaction []byte

	CreatedAt time.Time
	UpdatedAt time.Time
	State     BidState
}

type BidState string

const (
	BidStatePending  BidState = "pending"
	BidStateWon      BidState = "won"
	BidStateIncluded BidState = "included"
	BidStateReported BidState = "reported"
)

// TODO: combine CleanStore and VerifyBids

func CleanStore(ctx context.Context, store Store) error {
	log := log.New(log.Writer(), "CleanStore: ", log.Flags())

	// Some way to classify bids as being too old. Here we use wall clock time,
	// but it also be based on the current chain height.
	cutoff := time.Now().Add(-time.Minute)

	bids, err := store.ListBids(ctx)
	if err != nil {
		return fmt.Errorf("list bids: %w", err)
	}

	log.Printf("bid count %d", len(bids))

	for _, bid := range bids {
		switch bid.State {
		case BidStatePending:
			// Bids in a pending state past their auction have not won, and can
			// be deleted.
			if bid.CreatedAt.Before(cutoff) {
				store.DeleteBid(ctx, bid.ProposerAddress, bid.ChainID, bid.Height)
			}

		case BidStateWon:
			// Bids that received a commitment have won their auction. We have a
			// separate process that verifies winning bids are actually included
			// in the auction block, and marks the bid as either included or
			// ignored. So if a bid stays in the won state for too long, it
			// probably means that verification process isn't running correctly.
			if bid.UpdatedAt.Before(cutoff) {
				log.Printf("bid (%s@%d): winning bid was never verified, maybe a bug? -- deleting", bid.ChainID, bid.Height)
				store.DeleteBid(ctx, bid.ProposerAddress, bid.ChainID, bid.Height)
			}

		case BidStateIncluded:
			// Included bids are winning bids that have been verified to be part
			// of the auction block. We can delete them from the store.
			store.DeleteBid(ctx, bid.ProposerAddress, bid.ChainID, bid.Height)

		case BidStateReported:
			// Reported bids are winning bids that have been verified to NOT be
			// part of the auction block and for which we have already reported
			// the proposer. The code that does that verification
			// (and sets the ignored state on the bid) also gathers and submits
			// evidence, so we can just delete the bid here.
			store.DeleteBid(ctx, bid.ProposerAddress, bid.ChainID, bid.Height)

		default:
			log.Printf("bid (%s@%d): unknown bid state %q -- deleting", bid.ChainID, bid.Height, bid.State)
			store.DeleteBid(ctx, bid.ProposerAddress, bid.ChainID, bid.Height)
		}
	}

	return nil
}

func VerifyBids(
	ctx context.Context,
	store Store,
	node Node,
	builderKey *sdk_x_builder_types.Key,
) error {
	log := log.New(log.Writer(), "VerifyBids: ", log.Flags())

	currentHeight, err := node.CurrentHeight(ctx)
	if err != nil {
		return fmt.Errorf("get current height: %w", err)
	}

	log.Printf("current height %d", currentHeight)

	bids, err := store.ListBids(ctx)
	if err != nil {
		return fmt.Errorf("list bids: %w", err)
	}

	for _, bid := range bids {
		if bid.State != BidStateWon {
			continue
		}

		if currentHeight < bid.Height {
			continue
		}

		bidKey := fmt.Sprintf("(%s %s %d)", bid.ProposerAddress, bid.ChainID, bid.Height)
		txHash := hashOf(bid.SegmentCommitmentTransaction)
		err := node.VerifyTxInclusion(ctx, bid.Height, txHash)
		switch {
		case err == nil:
			bid.State = BidStateIncluded
			log.Printf("bid %s: winning bid verified", bidKey)

		case err != nil: // TODO: need to distinguish true-negative from false-negative
			log.Printf("bid %s: winning bid verification FAILED: %v", bidKey, err)

			info, err := node.AccountAtHeight(ctx, currentHeight, builderKey.Address)
			if err != nil {
				log.Printf("bid %s: account info fetching FAILED: %v", bidKey, err)
				break
			}

			reportTx, err := buildTx(ctx, builderKey, bid.ChainID, info.AccountNumber, info.Sequence,
				&sdk_x_builder_types.MsgReportProposer{
					BuilderAddress: builderKey.Address,
					Commitment:     *bid.SegmentCommitment,
				},
			)

			if err != nil {
				log.Printf("bid %s: report proposer %q failed: %v", bidKey, bid.SegmentCommitment.ProposerAddress, err)
				break
			}

			if _, err := node.BroadcastTxAsync(ctx, reportTx); err != nil {
				log.Printf("bid %s: report proposer %q failed: %v", bidKey, bid.SegmentCommitment.ProposerAddress, err)
				break
			}

			log.Printf("bid %s: reported proposer %q", bidKey, bid.SegmentCommitment.ProposerAddress)

			bid.State = BidStateReported
		}

		if err := store.UpdateBid(ctx, bid); err != nil {
			log.Printf("bid %s: update bid in store: error: %v", bidKey, err)
		}
	}

	return nil
}

//
// memStore
//

// bidKey is a struct used as the key type for the bids map.
type bidKey struct {
	proposerAddr string
	chainID      string
	height       int64
}

// memStore is a struct that implements the Store interface using an in-memory map.
type memStore struct {
	mu   sync.RWMutex
	bids map[bidKey]*Bid
}

// NewMemStore creates a new instance of memStore with an empty map of bids.
func NewMemStore() *memStore {
	return &memStore{
		bids: make(map[bidKey]*Bid),
	}
}

// UpdateBid updates an existing bid in the store.
func (s *memStore) UpdateBid(ctx context.Context, bid *Bid) error {
	now := time.Now().UTC()
	key := bidKey{proposerAddr: bid.ProposerAddress, chainID: bid.ChainID, height: bid.Height}

	s.mu.Lock()
	defer s.mu.Unlock()

	oldBid, ok := s.bids[key]
	if !ok {
		return fmt.Errorf("%w (chain=%s height=%d)", ErrBidNotFound, bid.ChainID, bid.Height)
	}

	if bid.SegmentCommitment != nil {
		oldBid.SegmentCommitment = bid.SegmentCommitment
	}

	if bid.SegmentCommitmentTransaction != nil {
		oldBid.SegmentCommitmentTransaction = bid.SegmentCommitmentTransaction
	}

	if bid.State != "" {
		oldBid.State = bid.State
	}

	oldBid.UpdatedAt = now

	return nil
}

// InsertBid inserts a new bid into the store.
func (s *memStore) InsertBid(ctx context.Context, bid *Bid) error {
	now := time.Now().UTC()
	bid.CreatedAt = now
	bid.UpdatedAt = now

	if err := validateBid(bid); err != nil {
		return fmt.Errorf("insert bid: %w", err)
	}

	key := bidKey{proposerAddr: bid.ProposerAddress, chainID: bid.ChainID, height: bid.Height}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.bids[key]; ok {
		return fmt.Errorf("bid already exists (chain=%s height=%d)", bid.ChainID, bid.Height)
	}

	s.bids[key] = bid

	return nil
}

// SelectBid retrieves a bid from the store by its chain ID and height.
func (s *memStore) SelectBid(ctx context.Context, proposerAddr, chainID string, height int64) (*Bid, error) {
	key := bidKey{proposerAddr: proposerAddr, chainID: chainID, height: height}

	s.mu.RLock()
	defer s.mu.RUnlock()

	bid, ok := s.bids[key]
	if !ok {
		return nil, ErrBidNotFound
	}

	dup := *bid
	return &dup, nil
}

// ListBids returns all bids in the store in sorted order by height ascending.
func (m *memStore) ListBids(ctx context.Context) ([]*Bid, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bids := make([]*Bid, 0, len(m.bids))
	for _, bid := range m.bids {
		dup := *bid
		bids = append(bids, &dup)
	}

	sort.Slice(bids, func(i, j int) bool {
		return bids[i].Height < bids[j].Height
	})

	return bids, nil
}

var ErrBidNotFound = errors.New("bid not found")

// DeleteBid removes the bid identified by chain ID and height from the store, if it exists.
func (s *memStore) DeleteBid(ctx context.Context, proposerAddr, chainID string, height int64) error {
	key := bidKey{proposerAddr: proposerAddr, chainID: chainID, height: height}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.bids[key]; !ok {
		return ErrBidNotFound
	}

	delete(s.bids, key)
	return nil
}

// validateBid is a helper function that validates the ChainID and Height fields of a bid.
func validateBid(bid *Bid) error {
	if bid.ChainID == "" {
		return fmt.Errorf("invalid chain ID")
	}

	if bid.Height == 0 {
		return fmt.Errorf("invalid height")
	}

	return nil
}
