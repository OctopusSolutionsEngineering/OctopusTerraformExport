package util

import (
	"strings"
)

func NilIfEmptyPointer(input *string) *string {
	if input == nil {
		return nil
	}

	if *input == "" {
		return nil
	}

	return input
}

func NilIfEmpty(input string) *string {
	if input == "" {
		return nil
	}

	return &input
}

func EmptyIfNil(input *string) string {
	if input == nil {
		return ""
	}

	return *input
}

func DefaultIfEmptyOrNil(input *string, defaultValue string) string {
	if input == nil || *input == "" {
		return defaultValue
	}

	return *input
}

func EnsureSuffix(input string, suffix string) string {
	if strings.HasSuffix(input, suffix) {
		return input
	}

	return input + suffix
}
