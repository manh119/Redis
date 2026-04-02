package core

import "github.com/manh119/Redis/internal/core/data_structure"

var dictStore *data_structure.Dictionary
var setStore map[string]data_structure.Set

func init() {
	dictStore = data_structure.NewDictionary()
	setStore = make(map[string]data_structure.Set)
}
