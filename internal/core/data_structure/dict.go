package data_structure

import (
	"errors"
	"time"

	"github.com/manh119/Redis/internal/core/config"
)

type Dictionary struct {
	dictStore        map[string]*Obj  // key -> value
	expiredDictStore map[string]int64 // key -> TTL
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
		dictStore:        make(map[string]*Obj),
		expiredDictStore: make(map[string]int64),
	}
}

func (dict *Dictionary) Get(key string) (string, error) {
	if dict.dictStore[key] == nil {
		return "", errors.New(config.NILL)
	}
	if dict.expiredDictStore[key] == -1 || dict.expiredDictStore[key] > time.Now().UnixMilli() {
		return dict.dictStore[key].Value, nil
	} else {
		delete(dict.dictStore, key)
		delete(dict.expiredDictStore, key)
		return "", errors.New(config.NILL)
	}
}

func (dict *Dictionary) Set(key string, value string, ttlInMs int64) {
	dict.dictStore[key] = NewObj(value)
	if ttlInMs < 0 {
		dict.expiredDictStore[key] = -1
	} else {
		dict.expiredDictStore[key] = time.Now().UnixMilli() + ttlInMs
	}
}

func (dict *Dictionary) Ttl(key string) int64 {
	if dict.dictStore[key] == nil {
		return -2
	}
	expiry, _ := dict.expiredDictStore[key]
	if expiry == -1 {
		return -1
	}
	now := time.Now().UnixMilli()
	diff := expiry - now

	if diff <= 0 {
		delete(dict.dictStore, key)
		delete(dict.expiredDictStore, key)
		return -2
	}

	return (diff + 999) / 1000
}

func (dict *Dictionary) Expire(key string, ttl int64) int {
	if dict.dictStore[key] == nil {
		return 0
	}
	if ttl == -1 {
		dict.expiredDictStore[key] = -1
	} else {
		dict.expiredDictStore[key] = time.Now().UnixMilli() + ttl
	}
	return 1
}

func (dict *Dictionary) Exists(args []string) int {
	count := 0
	for _, key := range args {
		if dict.dictStore[key] != nil &&
			((dict.expiredDictStore[key] > time.Now().UnixMilli()) || dict.expiredDictStore[key] == -1) {
			count++
		}
	}
	return count
}

func (dict *Dictionary) Del(args []string) int {
	count := 0
	for _, key := range args {
		if dict.dictStore[key] != nil &&
			((dict.expiredDictStore[key] > time.Now().UnixMilli()) || dict.expiredDictStore[key] == -1) {
			delete(dict.dictStore, key)
			delete(dict.expiredDictStore, key)
			count++
		}
	}
	return count
}
