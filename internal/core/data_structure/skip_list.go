package data_structure

import "math/rand"

const MaxLevel = 40
const Probability = 0.25

type SkipList struct {
	level         int
	header        *Node
	valueStoreMap map[string]float64
}

// forward[0] = next node in level 0, forward[1] = next node in level 1
type Node struct {
	value   string
	score   float64
	forward []*Node
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
	return &Node{
		value:   value,
		score:   score,
		forward: make([]*Node, levelNode+1),
	}
}

// sorted by score
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
		for current.forward[i] != nil && current.forward[i].score < score {
			current = current.forward[i]
		}
		update[i] = current
	}

	// 3. insert to all level
	sl.valueStoreMap[value] = score
	newNode := NewNode(value, score, levelInsert)
	for i := 0; i <= levelInsert; i++ {
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}
}

func (sl *SkipList) Search(score float64) *Node {
	current := sl.header
	// 			   1      ->      6	-> 7
	// skip list : 1 -> 3 -> 5 -> 6 -> 7
	// search node 5 -> try to find node 3 in level 0
	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].score < score {
			current = current.forward[i]
		}
	}

	// try compare next node in level 0
	if current != nil && current.forward[0].score == score {
		return current.forward[0]
	}
	return nil
}

func (sl *SkipList) getScore(value string) float64 {
	return sl.valueStoreMap[value]
}
