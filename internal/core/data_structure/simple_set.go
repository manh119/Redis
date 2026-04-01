package data_structure

type simple_set struct {
	values map[any]string // [value1, value2, value3]
}

func newSimpleSet() *simple_set {
	return &simple_set{
		values: make(map[any]string),
	}
}
