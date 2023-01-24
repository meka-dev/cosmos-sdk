package types

import (
	"encoding/binary"
)

var _ binary.ByteOrder

const (
	// BuilderStoreKeyPrefix is the prefix to retrieve all Builders
	BuilderStoreKeyPrefix = "Builder/value/"
)

// BuilderStoreKey returns the store key to retrieve a Builder from the address field
func BuilderStoreKey(address string) []byte {
	return []byte(address + "/")
}
