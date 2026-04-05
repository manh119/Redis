package data_structure

import (
	"math"

	"github.com/twmb/murmur3"
)

const Log10PointFive = -0.30102999566

// count min sketch
type CMS struct {
	depth   uint64     // row
	width   uint64     // col
	counter [][]uint64 // counter[row][col]
}

// assume errRate and errProb != 0
func NewCMS(errRate float64, errProb float64) *CMS {
	width, depth := CalcCMSDim(errRate, errProb)

	cms := &CMS{
		width:   width,
		depth:   depth,
		counter: make([][]uint64, depth),
	}

	for d := 0; uint64(d) < depth; d++ {
		cms.counter[d] = make([]uint64, width)
		for w := 0; uint64(w) < width; w++ {
			cms.counter[d][w] = 0
		}
	}

	return cms
}

// CalcCMSDim calculates the dimensions (width and depth) of the CMS
// based on the desired error rate and probability.
func CalcCMSDim(errRate float64, errProb float64) (uint64, uint64) {
	w := uint64(math.Ceil(2.0 / errRate))
	d := uint64(math.Ceil(math.Log10(errProb) / Log10PointFive))
	return w, d
}

func (cms *CMS) IncreaseBy(item string, increment uint64) uint64 {
	h1 := murmur3.SeedSum64(0, []byte(item))
	h2 := murmur3.SeedSum64(1, []byte(item))
	var count uint64 = math.MaxUint64

	for d := uint64(0); d < cms.depth; d++ {
		combinedHash := h1 + d*h2
		w := combinedHash % cms.width
		cms.counter[d][w] += increment
		val := cms.counter[d][w]
		if val < count {
			count = val
		}
	}

	return count
}

func (cms *CMS) Query(item string) uint64 {
	var count uint64 = math.MaxUint64
	h1 := murmur3.SeedSum64(0, []byte(item))
	h2 := murmur3.SeedSum64(1, []byte(item))

	// get min for low hash colision
	for d := uint64(0); d < cms.depth; d++ {
		combinedHash := h1 + d*h2
		w := combinedHash % cms.width

		val := cms.counter[d][w]
		if val < count {
			count = val
		}
	}
	return count
}
