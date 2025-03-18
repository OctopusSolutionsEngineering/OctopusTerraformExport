package hash

import (
	"crypto/sha256"
	"fmt"
)

// Sha256Hash returns the sha256 hash of the input string
func Sha256Hash(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	hashSum := hash.Sum(nil)
	return fmt.Sprintf("%x", hashSum)
}
