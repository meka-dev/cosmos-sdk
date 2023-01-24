package types

import (
	"bytes"
	"encoding/json"
	"fmt"

	sdk_crypto_types "github.com/cosmos/cosmos-sdk/crypto/types"
)

type jsonSegmentCommitment struct {
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
	SegmentBytes     int32    `json:"segment_bytes"`
	SegmentGas      int32    `json:"segment_gas"`
	SegmentHash     []byte   `json:"segment_hash"`
	PaymentPromise  string   `json:"payment_promise"`

	ProposerSignature []byte `json:"proposer_signature,omitempty"`
	BuilderSignature  []byte `json:"builder_signature,omitempty"`
}

func (sc SegmentCommitment) MarshalJSON() ([]byte, error) {
	return json.Marshal(&jsonSegmentCommitment{
		ProposerAddress:   sc.ProposerAddress,
		BuilderAddress:    sc.BuilderAddress,
		ChainID:           sc.ChainId,
		Height:            sc.Height,
		PreferenceIDs:     sc.PreferenceIds,
		PrefixOffset:      sc.PrefixOffset,
		PrefixLength:      sc.PrefixLength,
		PrefixHash:        sc.PrefixHash,
		SegmentOffset:     sc.SegmentOffset,
		SegmentLength:     sc.SegmentLength,
		SegmentBytes:       sc.SegmentBytes,
		SegmentGas:        sc.SegmentGas,
		SegmentHash:       sc.SegmentHash,
		PaymentPromise:    sc.PaymentPromise,
		ProposerSignature: sc.ProposerSignature,
		BuilderSignature:  sc.BuilderSignature,
	})
}

func (sc *SegmentCommitment) UnmarshalJSON(data []byte) error {
	var v jsonSegmentCommitment
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*sc = SegmentCommitment{
		ProposerAddress:   v.ProposerAddress,
		BuilderAddress:    v.BuilderAddress,
		ChainId:           v.ChainID,
		Height:            v.Height,
		PreferenceIds:     v.PreferenceIDs,
		PrefixOffset:      v.PrefixOffset,
		PrefixLength:      v.PrefixLength,
		PrefixHash:        v.PrefixHash,
		SegmentOffset:     v.SegmentOffset,
		SegmentLength:     v.SegmentLength,
		SegmentBytes:       v.SegmentBytes,
		SegmentGas:        v.SegmentGas,
		SegmentHash:       v.SegmentHash,
		PaymentPromise:    v.PaymentPromise,
		ProposerSignature: v.ProposerSignature,
		BuilderSignature:  v.BuilderSignature,
	}

	return nil
}

func (sc *SegmentCommitment) SignaturesHash() []byte {
	return HashByteSlices(sc.ProposerSignature, sc.BuilderSignature)
}

func (sc *SegmentCommitment) GetSignBytes() []byte {
	if sc == nil {
		return nil
	}

	cp := *sc
	cp.PreferenceIds = normalizeArray(cp.PreferenceIds)
	cp.PrefixHash = normalizeArray(cp.PrefixHash)
	cp.SegmentHash = normalizeArray(cp.SegmentHash)
	cp.ProposerSignature = nil
	cp.BuilderSignature = nil

	return signBytesEncoding(&cp)
}

func (sc *SegmentCommitment) VerifySignatures(builderPubKey, proposerPubKey sdk_crypto_types.PubKey) error {
	if !builderPubKey.VerifySignature(sc.GetSignBytes(), sc.BuilderSignature) {
		return fmt.Errorf("invalid builder signature")
	}

	if !proposerPubKey.VerifySignature(sc.GetSignBytes(), sc.ProposerSignature) {
		return fmt.Errorf("invalid proposer signature")
	}

	return nil
}

func (sc *SegmentCommitment) VerifyBlockHashes(txs [][]byte) error {
	prefix := txs[sc.PrefixOffset : sc.PrefixOffset+sc.PrefixLength]
	if !bytes.Equal(sc.PrefixHash, HashByteSlices(prefix...)) {
		return fmt.Errorf("invalid prefix hash") // TODO: Exported error value
	}

	segment := txs[sc.SegmentOffset : sc.SegmentOffset+sc.SegmentLength]
	if !bytes.Equal(sc.SegmentHash, HashByteSlices(segment...)) {
		return fmt.Errorf("invalid segment hash") // TODO: Exported error value
	}

	return nil
}
