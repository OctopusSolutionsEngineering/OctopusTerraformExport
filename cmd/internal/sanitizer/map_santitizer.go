package sanitizer

import (
	"fmt"
)

// SanitizeMap takes a map returned by the Octopus API, and replaces any sensitive value references with a placeholder
func SanitizeMap(input map[string]any) map[string]string {
	fixedMap := map[string]string{}
	for k, v := range input {
		if _, ok := v.(string); ok {
			fixedMap[k] = fmt.Sprintf("%v", v)
		} else {
			fixedMap[k] = "replace me with a password"
		}
	}
	return fixedMap
}
