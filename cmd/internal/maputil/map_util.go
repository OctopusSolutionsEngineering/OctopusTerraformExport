package maputil

import "github.com/samber/lo"

func NilIfEmptyMap(input map[string]string) *map[string]string {
	if len(input) == 0 {
		return nil
	}

	return &input
}

func ToStringAnyMap(input map[string]string) map[string]any {
	if input == nil {
		return nil
	}

	return lo.MapValues(input, func(value string, key string) any {
		return any(value)
	})
}

func ValueOrStringDefault(input map[string]any, key string, defaultValue string) string {
	if input == nil {
		return defaultValue
	}

	value, exists := input[key]
	if !exists {
		return defaultValue
	}

	if value == nil {
		return defaultValue
	}

	return value.(string)
}

func ValueOrBoolDefault(input map[string]any, key string, defaultValue bool) bool {
	if input == nil {
		return defaultValue
	}

	value, exists := input[key]
	if !exists {
		return defaultValue
	}

	if value == nil {
		return defaultValue
	}

	return value.(bool)
}
