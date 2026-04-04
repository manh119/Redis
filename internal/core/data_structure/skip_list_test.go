package data_structure

import (
	"fmt"
	"testing"
)

func TestSkipList_InsertAndSearch(t *testing.T) {
	sl := NewSkipList()

	// Dữ liệu mẫu
	data := []struct {
		val   string
		score float64
	}{
		{"A", 10.5},
		{"B", 20.0},
		{"C", 5.5},
		{"D", 30.2},
	}

	// Thực hiện chèn
	for _, d := range data {
		sl.Insert(d.val, d.score)
	}

	// Kiểm tra tìm kiếm các phần tử đã chèn
	for _, d := range data {
		node := sl.Search(d.score)
		if node == nil {
			t.Errorf("Expected to find node with value %s, but got nil", d.val)
		} else if node.value != d.val || node.score != d.score {
			t.Errorf("Expected node {value: %s, score: %f}, got {value: %s, score: %f}",
				d.val, d.score, node.value, node.score)
		}
	}
}

func TestSkipList_SearchNonExistent(t *testing.T) {
	sl := NewSkipList()
	sl.Insert("Exist", 100)

	tests := []float64{1, 2, 0}

	for _, val := range tests {
		node := sl.Search(val)
		if node != nil {
			t.Errorf("Expected nil for value %v, but got node with value %v", val, node.value)
		}
	}
}

func TestSkipList_Ordering(t *testing.T) {
	sl := NewSkipList()

	// Chèn không theo thứ tự
	scores := []float64{50, 10, 30, 20, 40}
	for _, s := range scores {
		sl.Insert("val", s)
	}

	// Duyệt tầng 0 từ header
	curr := sl.header.forward[0]
	lastScore := -1.0
	count := 0

	for curr != nil {
		if curr.score < lastScore {
			t.Errorf("Nodes are out of order: score %f appeared after %f", curr.score, lastScore)
		}
		lastScore = curr.score
		curr = curr.forward[0]
		count++
	}

	if count != len(scores) {
		t.Errorf("Expected %d nodes in list, but found %d", len(scores), count)
	}
}

func TestSkipList_Stress(t *testing.T) {
	sl := NewSkipList()
	n := 1000

	for i := 0; i < n; i++ {
		val := fmt.Sprintf("val%d", i)
		sl.Insert(val, float64(i))
	}

	// Kiểm tra ngẫu nhiên một vài phần tử
	target := 500
	node := sl.Search(float64(target))
	if node == nil || node.score != 500.0 {
		t.Errorf("Stress test failed: could not find target node")
	}
}
