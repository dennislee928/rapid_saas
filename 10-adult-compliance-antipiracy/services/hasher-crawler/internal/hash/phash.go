package hash

import (
	"encoding/binary"
	"hash/fnv"
	"math/bits"
)

// ImagePHash is a deterministic placeholder for a DCT pHash implementation.
// Production should replace this with real image decoding and perceptual hashing.
func ImagePHash(data []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(data)
	return h.Sum64()
}

// VideoFrameHashes returns one placeholder hash per supplied frame.
func VideoFrameHashes(frames [][]byte) []uint64 {
	hashes := make([]uint64, 0, len(frames))
	for _, frame := range frames {
		hashes = append(hashes, ImagePHash(frame))
	}
	return hashes
}

func HammingDistance(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

func Uint64Bytes(value uint64) []byte {
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, value)
	return out
}
