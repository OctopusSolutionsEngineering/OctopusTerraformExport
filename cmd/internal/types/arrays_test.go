package types

import "testing"

func TestIsArrayOrSlice(t *testing.T) {
	if !IsArrayOrSlice([]string{}) {
		t.Fatal("Should have returned true for slice")
	}

	if !IsArrayOrSlice([1]string{}) {
		t.Fatal("Should have returned true for array")
	}

	if IsArrayOrSlice("") {
		t.Fatal("Should have returned false for string")
	}

	if IsArrayOrSlice(1) {
		t.Fatal("Should have returned false for int")
	}

	if IsArrayOrSlice(nil) {
		t.Fatal("Should have returned false for nil")
	}

	if IsArrayOrSlice(true) {
		t.Fatal("Should have returned false for bool")
	}

	if IsArrayOrSlice(1.0) {
		t.Fatal("Should have returned false for float")
	}

	if IsArrayOrSlice(struct{}{}) {
		t.Fatal("Should have returned false for struct")
	}

	if IsArrayOrSlice(map[string]string{}) {
		t.Fatal("Should have returned false for map")
	}
}
