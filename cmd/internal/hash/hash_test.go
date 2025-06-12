package hash

import (
	"testing"
)

func TestSha256Hash(t *testing.T) {
	input := "your string here"
	expected := "ebea8483c5b21ae61081786be10f9704ce8975e1e5b505c03f6ab8514ecc5c0c" // Replace with the actual expected hash
	result := Sha256Hash(input)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}
