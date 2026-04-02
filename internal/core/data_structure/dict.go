package data_structure

import (
	"time"
)

type Dictionary struct {
	dictStore        map[string]any   // key -> value
	expiredDictStore map[string]int64 // key -> TTL
}

func NewDictionary() *Dictionary {
	return &Dictionary{
		dictStore:        make(map[string]any),
		expiredDictStore: make(map[string]int64),
	}
}

func (dict *Dictionary) Get(key string) any {
	if dict.dictStore[key] == nil {
		return "(nil)"
	}
	if dict.expiredDictStore[key] == -1 || dict.expiredDictStore[key] > time.Now().UnixMilli() {
		return dict.dictStore[key]
	} else {
		delete(dict.dictStore, key)
		delete(dict.expiredDictStore, key)
		return "(nil)"
	}
}

func (dict *Dictionary) Set(key string, value any, ttl int64) {
	dict.dictStore[key] = value
	if ttl < 0 {
		dict.expiredDictStore[key] = -1
	} else {
		dict.expiredDictStore[key] = time.Now().UnixMilli() + ttl
	}
}

func (dict *Dictionary) Ttl(key string) int64 {
	if dict.dictStore[key] == nil {
		return -2
	}
	if dict.expiredDictStore[key] == -1 {
		return -1
	}
	return (dict.expiredDictStore[key] - time.Now().UnixMilli()) / 1000
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
