package sanitizer

import (
	"regexp"
	"strings"
)

// SanitizeNamePointer creates a string pointer that can be used as a name for HCL resources
func SanitizeNamePointer(name *string) string {
	if name == nil {
		return ""
	}
	allowedChars := regexp.MustCompile(`[^A-Za-z0-9]`)
	return allowedChars.ReplaceAllString(strings.ToLower(*name), "_")
}

// SanitizeName creates a string that can be used as a name for HCL resources
func SanitizeName(name string) string {
	allowedChars := regexp.MustCompile(`[^A-Za-z0-9]`)
	return allowedChars.ReplaceAllString(strings.ToLower(name), "_")
}
