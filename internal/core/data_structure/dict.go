package data_structure

import (
	"errors"
	"log"
	"time"

	"github.com/manh119/Redis/internal/core"
	"github.com/manh119/Redis/internal/core/config"
)

type Dictionary struct {
	Dict        map[string]*Obj  // key -> value
	ExpiredDict map[string]int64 // key -> TTL
	NumberKey   int
}

type Obj struct {
	Value          string
	LastAccessTime uint32
}

func NewObj(value string) *Obj {
	return &Obj{
		Value:          value,
		LastAccessTime: now(),
	}
}

func now() uint32 {
	return uint32(time.Now().Unix())
}

func NewDictionary() *Dictionary {
	return &Dictionary{
		Dict:        make(map[string]*Obj),
		ExpiredDict: make(map[string]int64),
		NumberKey:   0,
	}
}

func (dict *Dictionary) Get(key string) (string, error) {
	if dict.Dict[key] == nil {
		return "", errors.New(config.NILL)
	}
	if dict.ExpiredDict[key] == -1 || dict.ExpiredDict[key] > time.Now().UnixMilli() {
		return dict.Dict[key].Value, nil
	} else {
		delete(dict.Dict, key)
		delete(dict.ExpiredDict, key)
		return "", errors.New(config.NILL)
	}
}

func (dict *Dictionary) Set(key string, value string, ttlInMs int64) {
	dict.Dict[key] = NewObj(value)

	dict.NumberKey = dict.NumberKey + 1
	if dict.NumberKey >= storage.MaxKeyNumber {
		dict.Evict()
	}

	if ttlInMs < 0 {
		dict.ExpiredDict[key] = -1
	} else {
		dict.ExpiredDict[key] = time.Now().UnixMilli() + ttlInMs
	}
}

func (dict *Dictionary) Ttl(key string) int64 {
	if dict.Dict[key] == nil {
		return -2
	}
	expiry, _ := dict.ExpiredDict[key]
	if expiry == -1 {
		return -1
	}
	now := time.Now().UnixMilli()
	diff := expiry - now

	if diff <= 0 {
		delete(dict.Dict, key)
		delete(dict.ExpiredDict, key)
		return -2
	}

	return (diff + 999) / 1000
}

func (dict *Dictionary) Expire(key string, ttl int64) int {
	if dict.Dict[key] == nil {
		return 0
	}
	if ttl == -1 {
		dict.ExpiredDict[key] = -1
	} else {
		dict.ExpiredDict[key] = time.Now().UnixMilli() + ttl
	}
	return 1
}

func (dict *Dictionary) Exists(args []string) int {
	count := 0
	for _, key := range args {
		if dict.Dict[key] != nil &&
			((dict.ExpiredDict[key] > time.Now().UnixMilli()) || dict.ExpiredDict[key] == -1) {
			count++
		}
	}
	return count
}

func (dict *Dictionary) Del(args []string) int {
	count := 0
	for _, key := range args {
		if dict.Dict[key] != nil &&
			((dict.ExpiredDict[key] > time.Now().UnixMilli()) || dict.ExpiredDict[key] == -1) {
			delete(dict.Dict, key)
			delete(dict.ExpiredDict, key)
			count++
		}
	}
	return count
}

func (dict *Dictionary) Evict() {
	switch storage.EvictionPolicy {
	case "allkeys-random":
		dict.evictRandom()
	case "allkeys-lru":
		dict.evictLru()
	}
}

func (dict *Dictionary) evictRandom() {
	evictCount := int64(storage.EvictionRatio * float64(storage.MaxKeyNumber))
	log.Print("trigger random eviction")
	for key := range dict.Dict {
		delete(dict.Dict, key)
		delete(dict.ExpiredDict, key)
		evictCount--
		if evictCount == 0 {
			break
		}
	}
}

func (dict *Dictionary) evictLru() {
	ePool.Evict()
}
