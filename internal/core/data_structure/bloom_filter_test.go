package data_structure

import (
	"testing"
)

func TestNewBloomFilter(t *testing.T) {
	errorRate := 0.01
	entries := uint64(1000)
	bf := NewBloomFilter(errorRate, entries)

	if bf.Entries != entries {
		t.Errorf("Expected entries %d, got %d", entries, bf.Entries)
	}
	if bf.bits == 0 {
		t.Error("Bits should not be zero")
	}
	if bf.hashes == 0 {
		t.Error("Hashes should not be zero")
	}
	if len(bf.bitset) == 0 {
		t.Error("Bitset slice should not be empty")
	}
}

func TestSetAndIsSet(t *testing.T) {
	bf := NewBloomFilter(0.01, 10)

	index := uint64(42)
	bf.SetBit(index)

	if !bf.IsSet(index) {
		t.Errorf("Bit at index %d should be set", index)
	}
}

func TestAddAndExist(t *testing.T) {
	bf := NewBloomFilter(0.001, 10000)

	testItems := []string{"apple", "banana", "cherry", "đồ sơn", "hà nội"}

	// Thêm các item vào filter
	for _, item := range testItems {
		bf.Add(item)
	}

	// Kiểm tra sự tồn tại (Tất cả phải True)
	for _, item := range testItems {
		if !bf.Exist(item) {
			t.Errorf("Item '%s' should exist in Bloom Filter", item)
		}
	}

	// Kiểm tra item chắc chắn không có
	if bf.Exist("watermelon") {
		// Có thể xảy ra False Positive, nhưng với 1000 entries và 0.01 rate,
		// xác suất cho 1 item ngẫu nhiên là rất thấp.
		t.Log("Note: False positive occurred for 'watermelon' (normal behavior but rare)")
	}
}

func TestFalsePositiveRate(t *testing.T) {
	entries := uint64(1000)
	errorRate := 0.01
	bf := NewBloomFilter(errorRate, entries)

	// Add 1000 items
	for i := 0; i < int(entries); i++ {
		bf.Add(string(rune(i)))
	}

	// Kiểm tra 1000 items khác để tính tỉ lệ false positive
	falsePositives := 0
	testCount := 1000
	for i := int(entries); i < int(entries)+testCount; i++ {
		if bf.Exist(string(rune(i))) {
			falsePositives++
		}
	}

	actualRate := float64(falsePositives) / float64(testCount)
	t.Logf("Actual False Positive Rate: %f", actualRate)

	if actualRate > errorRate*2 { // Cho phép sai số nhẹ trong thực tế
		t.Errorf("False positive rate too high: got %f, want ~%f", actualRate, errorRate)
	}
}
