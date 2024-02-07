package sanitizer

import (
	"regexp"
	"strings"
)

var allowedChars = regexp.MustCompile(`[^A-Za-z0-9]`)

// SanitizeNamePointer creates a string pointer that can be used as a name for HCL resources
func SanitizeNamePointer(name *string) string {
	if name == nil {
		return ""
	}
	return allowedChars.ReplaceAllString(strings.ToLower(*name), "_")
}

// SanitizeName creates a string that can be used as a name for HCL resources
func SanitizeName(name string) string {
	return allowedChars.ReplaceAllString(strings.ToLower(name), "_")
}

// SanitizeParameterName creates a string that can be used as slug in a step template parameter name
func SanitizeParameterName(name string) string {
	return allowedChars.ReplaceAllString(name, "")
}
