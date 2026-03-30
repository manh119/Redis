package core

import "github.com/manh119/Redis/internal/core/data_structure"

var dictStore *data_structure.Dictionary

func init() {
	dictStore = data_structure.NewDictionary()
}
