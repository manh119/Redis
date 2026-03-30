package data_structure

import (
	"time"
)

type Dictionary struct {
	dictStore        map[string]any    // key -> value
	expiredDictStore map[string]uint64 // key -> TTL
}

func NewDictionary() *Dictionary {
	return &Dictionary{}
}

func (dict *Dictionary) get(key string) any {
	if dict.dictStore[key] == nil {
		return nil
	}
	if dict.expiredDictStore[key] > uint64(time.Now().UnixMilli()) {
		return dict.dictStore[key]
	} else {
		delete(dict.dictStore, key)
		delete(dict.expiredDictStore, key)
		return nil
	}
}

func (dict *Dictionary) set(key string, value any, ttl int64) {
	dict.dictStore[key] = value
	dict.expiredDictStore[key] = uint64(time.Now().UnixMilli() + ttl)
}
