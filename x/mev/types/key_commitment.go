package types

const (
	// SegmentCommitmentStoreKeyPrefix is the prefix to retrieve all SegmentCommitments
	SegmentCommitmentStoreKeyPrefix = "SegmentCommitment/value/"
)

// SegmentCommitmentStoreKey returns the store key to retrieve a SegmentCommitment by its signatures hash
func SegmentCommitmentStoreKey(signaturesHash []byte) []byte {
	return signaturesHash
}
