package types_test

import (
	"bytes"
	"testing"

	sdk_x_builder_types "cosmossdk.io/x/mev/types"
)

func TestSignBytes(t *testing.T) {
	for _, testcase := range []struct {
		name string
		val  interface{ GetSignBytes() []byte }
		want []byte
	}{
		{
			name: "BidRequest full",
			val: &sdk_x_builder_types.BidRequest{
				ChainID:            "my-chain-id",
				Height:             123456,
				PreferenceIDs:      []string{"p1", "p3", "p2"},
				PaymentDenom:       "stake", // payment_denom should render before preference_ids
				PrefixTransactions: [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d"), []byte("e")},
				Signature:          []byte(`should not be included`),
			},
			want: []byte(`{"chain_id":"my-chain-id","height":"123456","payment_denom":"stake","preference_ids":["p1","p3","p2"],"prefix_transactions":["YQ==","Yg==","Yw==","ZA==","ZQ=="]}`),
		},
		{
			name: "BidRequest partial",
			val: &sdk_x_builder_types.BidRequest{
				ChainID:            "my-chain-id",
				Height:             123456,
				PreferenceIDs:      []string{}, // should render as `[]`
				PaymentDenom:       "",         //
				PrefixTransactions: nil,        // should also render as `[]` and not `null`
				Signature:          []byte{},
			},
			want: []byte(`{"chain_id":"my-chain-id","height":"123456","payment_denom":"","preference_ids":[],"prefix_transactions":[]}`),
		},
		{
			name: "BidRequest string escaping",
			val: &sdk_x_builder_types.BidRequest{
				ChainID:       "<script>HTML should not be escaped</script>",
				PaymentDenom:  `"quotes" should be escaped`,
				PreferenceIDs: []string{"Complex emoji 👨‍👨‍👦‍👦 and\nnewlines should be encoded correctly"},
			},
			want: []byte(`{"chain_id":"<script>HTML should not be escaped</script>","height":"0","payment_denom":"\"quotes\" should be escaped","preference_ids":["Complex emoji 👨‍👨‍👦‍👦 and\nnewlines should be encoded correctly"],"prefix_transactions":[]}`),
		},
		{
			name: "BidResponse full",
			val: &sdk_x_builder_types.BidResponse{
				ChainID:        "my-chain-id",
				Height:         123456,
				PreferenceIDs:  []string{"p1", "p3", "p2"},
				PrefixHash:     []byte("some hash"),
				PaymentPromise: "42stake",
				SegmentLength:  5,
				SegmentHash:    []byte("some other hash"),
				Signature:      []byte(`should not be included`),
			},
			want: []byte(`{"chain_id":"my-chain-id","height":"123456","payment_promise":"42stake","preference_ids":["p1","p3","p2"],"prefix_hash":"c29tZSBoYXNo","segment_hash":"c29tZSBvdGhlciBoYXNo","segment_length":5}`),
		},
		{
			name: "BidResponse partial",
			val: &sdk_x_builder_types.BidResponse{
				ChainID:        "my-chain-id",
				Height:         123456,
				PreferenceIDs:  nil, // should produce `"preference_ids":[]` -- `"preference_ids":null`
				PrefixHash:     nil,
				PaymentPromise: "",
				SegmentLength:  0,
				SegmentHash:    nil,
				Signature:      nil,
			},
			want: []byte(`{"chain_id":"my-chain-id","height":"123456","payment_promise":"","preference_ids":[],"prefix_hash":"","segment_hash":"","segment_length":0}`),
		},
		{
			name: "CommitRequest full",
			val: &sdk_x_builder_types.CommitRequest{
				ProposerAddress: "cosmos123456",
				ChainID:         "my-chain-id",
				Height:          123456,
				PrefixOffset:    0,
				PrefixLength:    0,
				PrefixHash:      []byte("default hash of no bytes"),
				SegmentOffset:   0,
				SegmentLength:   5,
				SegmentHash:     []byte("hash provided in bid response"),
				PaymentPromise:  "42stake",
				Signature:       []byte(`should not be included`),
			},
			want: []byte(`{"chain_id":"my-chain-id","height":"123456","payment_promise":"42stake","preference_ids":[],"prefix_hash":"ZGVmYXVsdCBoYXNoIG9mIG5vIGJ5dGVz","prefix_length":0,"prefix_offset":0,"proposer_address":"cosmos123456","segment_hash":"aGFzaCBwcm92aWRlZCBpbiBiaWQgcmVzcG9uc2U=","segment_length":5,"segment_offset":0}`),
		},
		{
			name: "CommitRequest partial",
			val: &sdk_x_builder_types.CommitRequest{
				ProposerAddress: "cosmos123456",
				ChainID:         "my-chain-id",
				Height:          123456,
				PrefixOffset:    0,
				PrefixLength:    0,
				PrefixHash:      nil,
				SegmentOffset:   0,
				SegmentLength:   0,
				SegmentHash:     nil,
				PaymentPromise:  "",
				Signature:       nil,
			},
			want: []byte(`{"chain_id":"my-chain-id","height":"123456","payment_promise":"","preference_ids":[],"prefix_hash":"","prefix_length":0,"prefix_offset":0,"proposer_address":"cosmos123456","segment_hash":"","segment_length":0,"segment_offset":0}`),
		},
		{
			name: "CommitResponse full",
			val: &sdk_x_builder_types.CommitResponse{
				ChainID:                      "my-chain-id",
				Height:                       123456,
				SegmentTransactions:          [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d"), []byte("e")},
				SegmentCommitmentTransaction: []byte("transaction data"),
				Signature:                    []byte(`should not be included`),
			},
			want: []byte(`{"segment_commitment_transaction":"dHJhbnNhY3Rpb24gZGF0YQ==","chain_id":"my-chain-id","height":"123456","segment_transactions":["YQ==","Yg==","Yw==","ZA==","ZQ=="]}`),
		},
		{
			name: "CommitResponse partial",
			val: &sdk_x_builder_types.CommitResponse{
				ChainID:                      "my-chain-id",
				Height:                       123456,
				SegmentTransactions:          [][]byte{{}, nil, []byte("c"), nil, {}},
				SegmentCommitmentTransaction: nil,
				Signature:                    nil,
			},
			want: []byte(`{"segment_commitment_transaction":"","chain_id":"my-chain-id","height":"123456","segment_transactions":["","","Yw==","",""]}`),
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			if want, have := testcase.want, testcase.val.GetSignBytes(); !bytes.Equal(want, have) {
				t.Errorf("bad SignBytes\n\twant: %s\n\thave: %s", string(want), string(have))
			}
		})

	}
}
