package types

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path"

	cometbft_abci_types "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk_crypto_keys_ed25119 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk_crypto_types "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
)

func HashByteSlices(vs ...[]byte) []byte {
	h := sha256.New()
	for _, v := range vs {
		h.Write(v)
	}
	return h.Sum(nil)
}

// Signer represents the signer the builder module signs requests with
// on behalf of block proposers, in order to authenticate those requests.
// Proposers must register such key ahead of time so that builder APIs
// are able to verify that signature.
type Signer interface {
	Sign(msg []byte) ([]byte, error)
}

// ProposerKey is the key that a proposer running the builder module
// uses to sign requests to external builders. Proposers must first register
// the address of the key and sign it with their operator key in order to
// establish a chain of trust.
type ProposerKey = Key

// Key encapsulates a private key and its corresponding pub key and Bech32 account address.
// It's used both for the proposer key in the builder module that signs external requests to
// builder APIs and, optionally, by builder APIs themselves when they are written in Go and
// choose to reuse the below code to load a builder key to sign responses.
//
// Builder APIs can export their builder keys with `networkd q builder export-json-key [address] [output-file]`.
type Key struct {
	PrivKey sdk_crypto_types.PrivKey `json:"priv_key"`
	PubKey  sdk_crypto_types.PubKey  `json:"pub_key"`
	Address string                   `json:"address"`
}

func (k *Key) Sign(msg []byte) ([]byte, error) {
	return k.PrivKey.Sign(msg)
}

func (k *Key) VerifySignature(msg, signature []byte) error {
	if !k.PubKey.VerifySignature(msg, signature) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

// LoadOrGenProposerKey attempts to load the ProposerKey from the given filePath. If
// the file does not exist, it generates and saves a new ProposerKey.
func LoadOrGenProposerKey(filePath string, cdc codec.Codec) (*ProposerKey, error) {
	if key, err := LoadKey(filePath, cdc); !os.IsNotExist(err) {
		return key, err
	}

	privKey := sdk_crypto_keys_ed25119.GenPrivKey()
	pubKey := privKey.PubKey()
	k := ProposerKey{
		PrivKey: privKey,
		PubKey:  pubKey,
		Address: sdk_types.AccAddress(pubKey.Address()).String(),
	}
	keyData, err := cdc.MarshalInterfaceJSON(k.PrivKey)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(path.Dir(filePath), 0700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filePath, keyData, 0600); err != nil {
		return nil, err
	}
	return &k, nil
}

func LoadKey(filePath string, cdc codec.Codec) (*Key, error) {
	keyData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var privkey sdk_crypto_types.PrivKey
	if err := cdc.UnmarshalInterfaceJSON(keyData, &privkey); err != nil {
		return nil, err
	}

	pubkey := privkey.PubKey()

	return &Key{
		PrivKey: privkey,
		PubKey:  pubkey,
		Address: sdk_types.AccAddress(pubkey.Address()).String(),
	}, nil
}

func (x *Proposer) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pubKey sdk_crypto_types.PubKey
	if err := unpacker.UnpackAny(x.Pubkey, &pubKey); err != nil {
		return err
	}
	if err := unpacker.UnpackAny(x.OperatorPubkey, &pubKey); err != nil {
		return err
	}
	return nil
}

func (x *Builder) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pubKey sdk_crypto_types.PubKey
	if err := unpacker.UnpackAny(x.Pubkey, &pubKey); err != nil {
		return err
	}
	return nil
}

func (x *QueryProposerResponse) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return x.Proposer.UnpackInterfaces(unpacker)
}

func (x *QueryProposersResponse) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	for i := range x.Proposers {
		if err := x.Proposers[i].UnpackInterfaces(unpacker); err != nil {
			return err
		}
	}
	return nil
}

func (x *QueryBuilderResponse) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return x.Builder.UnpackInterfaces(unpacker)
}

func (x *QueryBuildersResponse) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	for i := range x.Builders {
		if err := x.Builders[i].UnpackInterfaces(unpacker); err != nil {
			return err
		}
	}
	return nil
}

//
//
//

func PrepareProposalHandlerChain(handlers ...sdk_types.PrepareProposalHandler) sdk_types.PrepareProposalHandler {
	return func(sdkctx sdk_types.Context, req cometbft_abci_types.RequestPrepareProposal) cometbft_abci_types.ResponsePrepareProposal {
		var res cometbft_abci_types.ResponsePrepareProposal
		for _, h := range handlers {
			res = h(sdkctx, req)
			req.Txs = res.GetTxs()
		}
		return res
	}
}

func ProcessProposalHandlerChain(handlers ...sdk_types.ProcessProposalHandler) sdk_types.ProcessProposalHandler {
	return func(sdkctx sdk_types.Context, req cometbft_abci_types.RequestProcessProposal) cometbft_abci_types.ResponseProcessProposal {
		// TODO: not sure if this is correct?
		var res cometbft_abci_types.ResponseProcessProposal
		for _, h := range handlers {
			res = h(sdkctx, req)
			switch res.GetStatus() {
			case cometbft_abci_types.ResponseProcessProposal_ACCEPT:
				continue
			default:
				return res
			}
		}
		return res
	}
}
