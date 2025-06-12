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
