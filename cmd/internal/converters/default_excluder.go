package converters

import (
	"golang.org/x/exp/slices"
	"strings"
)

type DefaultExcluder struct {
}

func (e DefaultExcluder) IsResourceExcluded(resourceName string, excludeAll bool, excludeThese []string, excludeAllButThese []string) bool {
	if strings.TrimSpace(resourceName) == "" {
		return true
	}

	if excludeAll {
		return true
	}

	if excludeThese != nil && slices.Index(excludeThese, resourceName) != -1 {
		return true
	}

	if excludeAllButThese != nil && len(excludeAllButThese) != 0 && slices.Index(excludeAllButThese, resourceName) == -1 {
		return true
	}

	return false
}
