package core

import "github.com/manh119/Redis/internal/core/data_structure"

var DictStore *data_structure.Dictionary
var SetStore map[string]*data_structure.Set
var SkipListStore map[string]*data_structure.SkipList
var CmsStore map[string]*data_structure.CMS
var BloomFilterStore map[string]*data_structure.BloomFilter

func init() {
	InitStorage()
}

func InitStorage() {
	DictStore = data_structure.NewDictionary()
	SetStore = make(map[string]*data_structure.Set)
	SkipListStore = make(map[string]*data_structure.SkipList)
	CmsStore = make(map[string]*data_structure.CMS)
	BloomFilterStore = make(map[string]*data_structure.BloomFilter)
}
