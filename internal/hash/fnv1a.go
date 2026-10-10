package hash

const (
	fnvOffsetBasis uint64 = 14695981039346656037
	fnvPrime       uint64 = 1099511628211
)

// seed is xored into the offset basis so each seed acts as a different hash fn,
// seed 0 is plain FNV-1a
func FNV1a(seed uint64, data []byte) uint64 {
	h := fnvOffsetBasis ^ seed
	for _, b := range data {
		h ^= uint64(b)
		h *= fnvPrime
	}
	return h
}
