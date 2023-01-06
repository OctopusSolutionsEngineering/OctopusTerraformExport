package util

import (
	"regexp"
	"strings"
)

func SanitizeNamePointer(name *string) string {
	if name == nil {
		return ""
	}
	allowedChars := regexp.MustCompile(`[^A-Za-z0-9]`)
	return allowedChars.ReplaceAllString(strings.ToLower(*name), "_")
}

func SanitizeName(name string) string {
	allowedChars := regexp.MustCompile(`[^A-Za-z0-9]`)
	return allowedChars.ReplaceAllString(strings.ToLower(name), "_")
}
