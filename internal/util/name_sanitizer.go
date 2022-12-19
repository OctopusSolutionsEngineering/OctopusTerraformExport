package util

import (
	"regexp"
	"strings"
)

func SanitizeName(name string) string {
	allowedChars := regexp.MustCompile(`[^A-Za-z0-9]`)
	return allowedChars.ReplaceAllString(strings.ToLower(name), "_")
}
