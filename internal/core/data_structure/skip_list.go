package data_structure

import (
	"errors"
	"math/rand"
	"strings"
)

const MaxLevel = 40
const Probability = 0.25

type SkipList struct {
	level         int // level 0-based index
	header        *Node
	valueStoreMap map[string]float64
}

// forward[0] = next node in level 0, forward[1] = next node in level 1
// span = distance to the next node at a level
// span[0] = span at level 0 is alway is 1
type Node struct {
	value   string
	score   float64
	forward []*Node
	span    []int
}

// level in 0-based index
func NewSkipList() *SkipList {
	return &SkipList{
		level:         0,
		header:        NewNode("", 0, MaxLevel), // virtual header
		valueStoreMap: make(map[string]float64),
	}
}

func NewNode(value string, score float64, levelNode int) *Node {
	newNode := &Node{
		value:   value,
		score:   score,
		forward: make([]*Node, levelNode+1),
		span:    make([]int, levelNode+1),
	}
	newNode.span[0] = 1
	return newNode
}

// sorted by score
// if score is equal, sort by value
func (sl *SkipList) Insert(value string, score float64) {
	// 1. find level of new node by random
	levelInsert := 0
	for rand.Float64() < Probability && levelInsert < MaxLevel {
		levelInsert++
	}
	if levelInsert > sl.level {
		sl.level = levelInsert
	}

	// 2. search list position to insert
	// skip list : 1 -> 3 -> 5 -> 6
	// insert score 4 -> postion to insert = node 3
	update := make([]*Node, sl.level+1)
	current := sl.header
	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && (current.forward[i].score < score ||
			(current.forward[i].score == score && strings.Compare(current.forward[i].value, value) == -1)) {
			current = current.forward[i]
		}
		update[i] = current
	}

	// 3. insert to all level
	sl.valueStoreMap[value] = score
	newNode := NewNode(value, score, levelInsert)
	for i := 0; i <= levelInsert; i++ {
		updateSpan := update[i].span[i]
		update[i].span[i] = (updateSpan + 1) / 2
		newNode.span[i] = updateSpan + 1 - update[i].span[i]

		// update link node
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}
}

// return nill if don't exist value in skipList
func (sl *SkipList) Search(value string) *Node {
	score := sl.valueStoreMap[value]
	current := sl.header
	// 			   1      ->      6	-> 7
	// skip list : 1 -> 3 -> 5 -> 6 -> 7
	// search node 5 -> try to find node 3 in level 0
	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && (current.forward[i].score < score ||
			(current.forward[i].score == score && strings.Compare(current.forward[i].value, value) == -1)) {
			current = current.forward[i]
		}
	}

	// try compare next node in level 0
	if current != nil && current.forward[0].score == score {
		return current.forward[0]
	}
	return nil
}

func (sl *SkipList) GetRank(value string) (int, error) {
	score, exist := sl.valueStoreMap[value]
	if !exist {
		return -1, nil
	}
	current := sl.header
	rank := 0
	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && (current.forward[i].score <= score ||
			(current.forward[i].score == score && strings.Compare(current.forward[i].value, value) == 0)) {
			rank += current.forward[i].span[i]
			if current.forward[i].value == value {
				return rank - 1, nil
			}
			current = current.forward[i]
		}
	}

	return -1, nil
}

func (sl *SkipList) GetScore(value string) (float64, error) {
	score, exist := sl.valueStoreMap[value]
	if !exist {
		return 0, errors.New("value not exist")
	}
	return score, nil
}

func (sl *SkipList) Delete(value string) (int, error) {
	score, exist := sl.valueStoreMap[value]
	if !exist {
		return 0, nil
	}

	// find all ref in all level need to delete
	update := make([]*Node, sl.level+1)
	current := sl.header
	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && (current.forward[i].score < score ||
			(current.forward[i].score == score && strings.Compare(current.forward[i].value, value) == -1)) {
			current = current.forward[i]
		}
		update[i] = current
	}

	// not exist node to delete
	if current == nil || current.forward[0].score != score || current.forward[0].value != value {
		return 0, nil
	}

	// delete to all level
	delete(sl.valueStoreMap, value)
	for i := 0; i <= sl.level; i++ {
		deletedNode := update[i].forward[i]
		update[i].span[i] = update[i].span[i] + deletedNode.span[i] - 1
		update[i].forward[i] = deletedNode.forward[i]
	}
	return 1, nil
}
