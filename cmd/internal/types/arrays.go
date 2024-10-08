package types

import "reflect"

func IsArrayOrSlice(v interface{}) bool {
	if v == nil {
		return false
	}

	kind := reflect.TypeOf(v).Kind()
	return kind == reflect.Array || kind == reflect.Slice
}
