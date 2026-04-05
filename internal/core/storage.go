package core

import "github.com/manh119/Redis/internal/core/data_structure"

var dictStore *data_structure.Dictionary
var setStore map[string]*data_structure.Set
var skipListStore map[string]*data_structure.SkipList
var cmsStore map[string]*data_structure.CMS

func init() {
	InitStorage()
}

func InitStorage() {
	dictStore = data_structure.NewDictionary()
	setStore = make(map[string]*data_structure.Set)
	skipListStore = make(map[string]*data_structure.SkipList)
	cmsStore = make(map[string]*data_structure.CMS)
}
