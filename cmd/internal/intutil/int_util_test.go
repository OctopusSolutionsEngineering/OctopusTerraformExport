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

func TestParseIntPointer(t *testing.T) {
	// Test case: input is nil
	var inputNil *string
	result, err := ParseIntPointer(inputNil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}

	// Test case: input is a valid integer string
	inputValid := "123"
	result, err = ParseIntPointer(&inputValid)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || *result != 123 {
		t.Fatalf("expected 123, got %v", result)
	}

	// Test case: input is an invalid integer string
	inputInvalid := "abc"
	result, err = ParseIntPointer(&inputInvalid)
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}
