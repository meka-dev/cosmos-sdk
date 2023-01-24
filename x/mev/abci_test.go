package mev_test

import (
	"testing"

	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_crypto_keys_secp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
)

func TestSignBytesStable(t *testing.T) {
	t.Run("BidRequest", func(t *testing.T) {
		m := map[string]int{}
		for i := 0; i < 100; i++ {
			v := &sdk_x_builder_types.BidRequest{
				ChainID:            "turbochain-99",
				Height:             123,
				PrefixTransactions: [][]byte{},
				PreferenceIDs:      []string{"ofac-compliance", "no-frontruns"},
			}
			b := v.GetSignBytes()
			s := string(b)
			t.Log(s)
			m[s]++
		}
		if len(m) != 1 {
			t.Fatalf("cardinality %d", len(m))
		}
	})
}

func TestSigning(t *testing.T) {
	privkey := sdk_crypto_keys_secp256k1.GenPrivKey()
	pubkey := privkey.PubKey()

	t.Run("BidRequest", func(t *testing.T) {
		v := &sdk_x_builder_types.BidRequest{
			ChainID:            "turbochain-99",
			Height:             123,
			PrefixTransactions: [][]byte{},
			PreferenceIDs:      []string{"ofac-compliance", "no-frontruns"},
		}
		if err := v.SignWith(privkey); err != nil {
			t.Fatal(err)
		}
		t.Logf("signature %x", v.Signature)
		if !v.VerifySignature(pubkey) {
			t.Fatalf("invalid signature")
		}
	})
}
