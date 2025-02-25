package converters

import (
	"github.com/samber/lo"
	"regexp"
	"strings"
)

// Regex that matches Octostache and script functions referencing variables
var variableRe = regexp.MustCompile(`#\{.*?}|\$OctopusParameters\[.*?]|Octopus.Parameters\[.*?]|get_octopusvariable ".*?"|get_octopusvariable\(.*?\)`)

func LimitAttributeLength(length int, retainVariables bool, input string) string {
	if length <= 0 {
		return input
	}

	if len(input) > length {
		sanitizedValue := input[0 : length-1]
		if retainVariables {
			matches := variableRe.FindAllString(input, -1)
			if len(matches) > 0 {
				// only paste the unique variables that appear in the rest of the script
				sanitizedValue += " " + strings.Join(lo.Uniq(matches), " ")
			}
		}

		return sanitizedValue

	}

	return input
}
