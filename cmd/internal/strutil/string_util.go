package strutil

import (
	"regexp"
	"strconv"
	"strings"
)

func StrPointer(input string) *string {
	return &input
}

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

func FalseIfNil(input *bool) bool {
	if input == nil {
		return false
	}

	return *input
}

func NilIfFalse(input bool) *bool {
	if !input {
		return nil
	}

	return &input
}

func DefaultIfEmpty(input string, defaultValue string) string {
	if input == "" {
		return defaultValue
	}

	return input
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

func ParseBool(input string) bool {
	value, err := strconv.ParseBool(input)

	if err != nil {
		return false
	}

	return value
}

func ParseBoolPointer(input *string) *bool {
	if input == nil {
		return nil
	}

	value, err := strconv.ParseBool(*input)

	if err != nil {
		retValue := false
		return &retValue
	}

	return &value
}

func UnEscapeDollar(fileMap map[string]string) map[string]string {
	// Unescape dollar signs because of https://github.com/hashicorp/hcl/issues/323
	regex := regexp.MustCompile(`\$\$\{(.*?)}`)
	for k, v := range fileMap {
		fileMap[k] = regex.ReplaceAllString(v, "${$1}")
	}

	return fileMap
}
