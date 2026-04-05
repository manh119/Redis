package data_structure

import (
	"math"

	"github.com/twmb/murmur3"
)

type BloomFilter struct {
	errorRate float64
	hashes    uint64 // number of hash func
	bitset    []uint64
	Entries   uint64 // number of element
	bits      uint64 // number of bit
}

/*
http://en.wikipedia.org/wiki/Bloom_filter
- Optimal number of bits is: bits = (entries * ln(error)) / ln(2)^2
- bitPerEntry = bits/entries
- Optimal number of hash functions is: hashes = bitPerEntry * ln(2)
*/
func NewBloomFilter(errorRate float64, entries uint64) *BloomFilter {
	bits := uint64(math.Ceil(math.Abs(float64(entries) * math.Log(errorRate) / (math.Log(2) * math.Log(2)))))
	hashes := uint64(math.Ceil(math.Log(2) * float64(bits) / float64(entries)))
	size := uint64(math.Ceil(float64(bits) / 64.0))

	return &BloomFilter{
		errorRate: errorRate,
		hashes:    hashes,
		bits:      bits,
		Entries:   entries,
		bitset:    make([]uint64, size),
	}
}

// size 3 of []bits of type uint4 like
// word  in array :     0           1          2
// bit   in array :	[0 0 0 0] , [0 0 0 0], [0 0 0 0]
func (bf *BloomFilter) SetBit(index uint64) {
	index = index % bf.bits // handle case over-flow bit

	word := index / 64
	bit := index % 64
	bf.bitset[word] = bf.bitset[word] | (1 << bit)
}

func (bf *BloomFilter) IsSet(index uint64) bool {
	index = index % bf.bits

	word := index / 64
	bit := index % 64

	isSet := bf.bitset[word] & (1 << bit)
	return isSet == 1
}

func (bf *BloomFilter) Add(item string) {
	h1 := murmur3.SeedSum64(0, []byte(item))
	h2 := murmur3.SeedSum64(1, []byte(item))

	for hashFunc := uint64(0); hashFunc < bf.hashes; hashFunc++ {
		combinedHash := h1 + hashFunc*h2
		bf.SetBit(combinedHash)
	}
}

func (bf *BloomFilter) Exist(item string) bool {
	h1 := murmur3.SeedSum64(0, []byte(item))
	h2 := murmur3.SeedSum64(1, []byte(item))

	for hashFunc := uint64(0); hashFunc < bf.hashes; hashFunc++ {
		combinedHash := h1 + hashFunc*h2
		if !bf.IsSet(combinedHash) {
			return false
		}
	}
	return true
}
