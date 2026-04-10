package data_structure

import (
	"sort"

	"github.com/manh119/Redis/internal/config"
)

var EpoolMaxSize = 20
var ePool *EvictionPool = NewEvictionPool(0)

// Approximated Least recently used
type EvictionCandidate struct {
	key            string
	lastAccessTime uint32
}

type EvictionPool struct {
	pool []*EvictionCandidate
}

func NewEvictionPool(size int) *EvictionPool {
	return &EvictionPool{
		pool: make([]*EvictionCandidate, size),
	}
}

// push to EvictionPool + sort by access time + remove if exceed size
func (ep *EvictionPool) Push(key string, lastAccessTime uint32) {
	newItem := &EvictionCandidate{
		key:            key,
		lastAccessTime: lastAccessTime,
	}
	ep.pool = append(ep.pool, newItem)
	sort.Sort(ByLastAccessTime(ep.pool))

	if len(ep.pool) > config.EpoolMaxSize {
		lastIndex := len(ep.pool) - 1
		ep.pool = ep.pool[:lastIndex]
	}
}

// returns the oldest item in the pool
func (p *EvictionPool) Pop() *EvictionCandidate {
	if len(p.pool) == 0 {
		return nil
	}
	oldestItem := p.pool[0]
	p.pool = p.pool[1:]
	return oldestItem
}

type ByLastAccessTime []*EvictionCandidate

func (a ByLastAccessTime) Len() int {
	return len(a)
}

func (a ByLastAccessTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByLastAccessTime) Less(i, j int) bool {
	return a[i].lastAccessTime < a[j].lastAccessTime
}
