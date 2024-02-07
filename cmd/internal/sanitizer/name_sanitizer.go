package sanitizer

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
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
// It ensures the name is unique is the set of dependencies. This is important because a sanitized string can
// produce conflicts e.g. "DockerHub" and "Docker Hub" are both sanitized to "DockerHub".
func SanitizeParameterName(dependencies *data.ResourceDetailsCollection, name string, parameterType string) string {
	sanitizedName := allowedChars.ReplaceAllString(name, "")

	count := 1
	for _, r := range dependencies.Resources {
		for _, p := range r.Parameters {
			if strings.HasPrefix(p.ResourceName, sanitizedName) && p.ParameterType == parameterType {
				count = count + 1
			}
		}
	}

	// Append a count suffix for any subsequent conflicts
	if count != 1 {
		return sanitizedName + fmt.Sprint(count)
	}

	return sanitizedName
}
