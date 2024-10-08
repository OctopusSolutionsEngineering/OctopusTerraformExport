package types

import "reflect"

func IsArrayOrSlice(v interface{}) bool {
	kind := reflect.TypeOf(v).Kind()
	return kind == reflect.Array || kind == reflect.Slice
}
