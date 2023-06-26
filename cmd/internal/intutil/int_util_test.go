package intutil

import "testing"

func TestNilIfEmptyPointer(t *testing.T) {
	number := 5

	if ZeroIfNil(&number) != number {
		t.Fatalf("result should have been the same number")
	}

	if ZeroIfNil(nil) != 0 {
		t.Fatalf("result should have been zero")
	}
}
