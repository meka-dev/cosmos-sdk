package types

import (
	"bytes"
	"encoding/json"
	"strings"

	sdk_crypto_types "github.com/cosmos/cosmos-sdk/crypto/types"
)

// TODO: technically all types here should be versioned

type BidRequest struct {
	ProposerAddress    string   `json:"proposer_address"`
	ChainID            string   `json:"chain_id"`
	Height             int64    `json:"height,string"`
	PaymentDenom       string   `json:"payment_denom"`
	PreferenceIDs      []string `json:"preference_ids"`
	PrefixTransactions [][]byte `json:"prefix_transactions"`
	MaxBytes           int64    `json:"max_bytes"`
	MaxGas             int64    `json:"max_gas"`

	Signature []byte `json:"signature,omitempty"`
}

func (req *BidRequest) GetSignBytes() []byte {
	cp := *req
	cp.PreferenceIDs = normalizeArray(cp.PreferenceIDs)
	cp.PrefixTransactions = normalizeNestedArray(cp.PrefixTransactions)
	cp.Signature = nil
	return signBytesEncoding(cp)
}

func (req *BidRequest) SignWith(s Signer) error {
	sig, err := s.Sign(req.GetSignBytes())
	if err != nil {
		return err
	}
	req.Signature = sig
	return nil
}

func (req *BidRequest) VerifySignature(pubkey sdk_crypto_types.PubKey) bool {
	return pubkey.VerifySignature(req.GetSignBytes(), req.Signature)
}

//
//
//

type BidResponse struct {
	ProposerAddress string   `json:"proposer_address"`
	ChainID         string   `json:"chain_id"`
	Height          int64    `json:"height,string"`
	PreferenceIDs   []string `json:"preference_ids"`
	PrefixHash      []byte   `json:"prefix_hash"`
	PaymentPromise  string   `json:"payment_promise"`
	SegmentLength   int      `json:"segment_length"`
	SegmentBytes    int64    `json:"segment_bytes"`
	SegmentGas      int64    `json:"segment_gas"`
	SegmentHash     []byte   `json:"segment_hash"`

	Signature []byte `json:"signature,omitempty"`
}

func (res *BidResponse) GetSignBytes() []byte {
	cp := *res
	cp.PreferenceIDs = normalizeArray(cp.PreferenceIDs)
	cp.PrefixHash = normalizeArray(cp.PrefixHash)
	cp.SegmentHash = normalizeArray(cp.SegmentHash)
	cp.Signature = nil
	return signBytesEncoding(cp)
}

func (res *BidResponse) SignWith(s Signer) error {
	sig, err := s.Sign(res.GetSignBytes())
	if err != nil {
		return err
	}
	res.Signature = sig
	return nil
}

func (res *BidResponse) VerifySignature(pubKey sdk_crypto_types.PubKey) bool {
	return pubKey.VerifySignature(res.GetSignBytes(), res.Signature)
}

//
//
//

type CommitRequest struct {
	ProposerAddress string   `json:"proposer_address"`
	BuilderAddress  string   `json:"builder_address"`
	ChainID         string   `json:"chain_id"`
	Height          int64    `json:"height,string"`
	PreferenceIDs   []string `json:"preference_ids"`
	PrefixOffset    int32    `json:"prefix_offset"`
	PrefixLength    int32    `json:"prefix_length"`
	PrefixHash      []byte   `json:"prefix_hash"`
	SegmentOffset   int32    `json:"segment_offset"`
	SegmentLength   int32    `json:"segment_length"`
	SegmentBytes    int64    `json:"segment_bytes"`
	SegmentGas      int64    `json:"segment_gas"`
	SegmentHash     []byte   `json:"segment_hash"`
	PaymentPromise  string   `json:"payment_promise"`

	Signature []byte `json:"signature,omitempty"`
}

func (req *CommitRequest) GetSignBytes() []byte {
	cp := *req
	cp.PreferenceIDs = normalizeArray(cp.PreferenceIDs)
	cp.PrefixHash = normalizeArray(cp.PrefixHash)
	cp.SegmentHash = normalizeArray(cp.SegmentHash)
	cp.Signature = nil
	return signBytesEncoding(cp)
}

func (req *CommitRequest) SignWith(s Signer) error {
	sig, err := s.Sign(req.GetSignBytes())
	if err != nil {
		return err
	}
	req.Signature = sig
	return nil
}

func (req *CommitRequest) VerifySignature(pubKey sdk_crypto_types.PubKey) bool {
	return pubKey.VerifySignature(req.GetSignBytes(), req.Signature)
}

//
//
//

type CommitResponse struct {
	ChainID                      string   `json:"chain_id"`
	Height                       int64    `json:"height,string"`
	SegmentTransactions          [][]byte `json:"segment_transactions"`
	SegmentCommitmentTransaction []byte   `json:"segment_commitment_transaction"`

	Signature []byte `json:"signature,omitempty"`
}

func (res *CommitResponse) GetSignBytes() []byte {
	cp := *res
	cp.SegmentTransactions = normalizeNestedArray(cp.SegmentTransactions)
	cp.SegmentCommitmentTransaction = normalizeArray(cp.SegmentCommitmentTransaction)
	cp.Signature = nil
	return signBytesEncoding(cp)
}

func (res *CommitResponse) SignWith(s Signer) error {
	sig, err := s.Sign(res.GetSignBytes())
	if err != nil {
		return err
	}
	res.Signature = sig
	return nil
}

func (res *CommitResponse) VerifySignature(pubKey sdk_crypto_types.PubKey) bool {
	return pubKey.VerifySignature(res.GetSignBytes(), res.Signature)
}

//
//
//

func signBytesEncoding(val any) []byte {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)

	buf.Reset()
	if err := enc.Encode(val); err != nil {
		panic(err)
	}

	var intermediate map[string]any
	if err := json.Unmarshal(buf.Bytes(), &intermediate); err != nil {
		panic(err)
	}

	for k := range intermediate {
		if strings.HasSuffix(k, "signature") {
			delete(intermediate, k) // just to be safe
		}
	}

	buf.Reset()
	if err := enc.Encode(intermediate); err != nil {
		panic(err)
	}

	return bytes.TrimSpace(buf.Bytes()) // Encode adds a trailing newline we need to remove
}

func normalizeArray[T any](field []T) []T {
	if field == nil {
		return []T{}
	}
	return field
}

func normalizeNestedArray[T any](field [][]T) [][]T {
	if field == nil {
		return [][]T{}
	}
	for i, inner := range field {
		field[i] = normalizeArray(inner)
	}
	return field
}
