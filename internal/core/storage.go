package core

import "github.com/manh119/Redis/internal/core/data_structure"

var dictStore *data_structure.Dict

func init() {
	dictStore = data_structure.CreateDict()
}
