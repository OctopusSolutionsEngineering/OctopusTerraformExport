package strutil

import (
	"regexp"
	"strconv"
	"strings"
)

var regex = regexp.MustCompile(`"\$\$\{([^}]*)}"`)

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

func EmptyPointerIfNil(input *string) *string {
	empty := ""

	if input == nil {
		return &empty
	}

	return input
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

func InputPointerIfEnabled(enabled bool, input *string) *string {
	if enabled {
		return input
	}

	return nil
}

func InputIfEnabled(enabled bool, input string) *string {
	if enabled {
		return &input
	}

	return nil
}

func InputIfEnabledElseDefault(enabled bool, input string, defaultValue string) *string {
	if enabled {
		return &input
	}

	return &defaultValue
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

// UnEscapeDollarInMap is a naive way of unescaping strings that assumes any string whose entire
// contents is two dollar signs, an opening curly bracket, some content, and a closing curly bracket
// was meant to be a HCL interpolated string.
// Where this assumption doesn't hold, converters must write attributes manually rather than rely on
// this method. See ProjectConverter for an example where the description field is written out manually.
func UnEscapeDollarInMap(fileMap map[string]string) map[string]string {
	for k, v := range fileMap {
		fileMap[k] = UnEscapeDollar(v)
	}

	return fileMap
}

func UnEscapeDollar(input string) string {
	// Unescape dollar signs because of https://github.com/hashicorp/hcl/issues/323
	return regex.ReplaceAllString(input, "\"${$1}\"")
}
