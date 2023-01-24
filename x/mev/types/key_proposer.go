package types

const (
	// ProposerStoreKeyPrefix is the prefix to retrieve all Proposer
	ProposerStoreKeyPrefix = "Proposer/value/"
	// ProposerStoreOperatorAddressIndexPrefix is the prefix to retrieve a Proposer
	// address by its operator address.
	ProposerStoreOperatorAddressIndexPrefix = "Proposer/operator-addr/"
	// ProposerInfractionStoreKeyPrefix is the prefix to retrieve a proposer's infractions
	ProposerInfractionStoreKeyPrefix = "ProposerInfraction/"
)

// ProposerStoreKey returns the store key to retrieve a Proposer from the address field
func ProposerStoreKey(address string) []byte {
	return []byte(address + "/")
}

func ProposerByOperatorAddressKey(address string) []byte {
	return []byte(address + "/")
}

func ProposerInfractionStoreKey(address string, signaturesHash []byte) []byte {
	return append([]byte(address+"/"), signaturesHash...)
}
