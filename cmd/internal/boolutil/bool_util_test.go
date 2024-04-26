package boolutil

import "testing"

func TestFalseIfNilBool(t *testing.T) {
	boolean := true

	if !FalseIfNil(&boolean) {
		t.Fatalf("result should have been true ")
	}

	if FalseIfNil(nil) {
		t.Fatalf("result should have been false")
	}
}
