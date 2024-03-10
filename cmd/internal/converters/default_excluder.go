package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
	"regexp"
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

	if excludeAllButThese != nil && len(excludeAllButThese) != 0 {
		// Ignore any empty strings
		filteredList := lo.Filter(excludeAllButThese, func(item string, index int) bool {
			return strings.TrimSpace(item) != ""
		})

		if len(filteredList) != 0 && slices.Index(filteredList, resourceName) == -1 {
			return true
		}
	}

	return false
}

func (e DefaultExcluder) IsResourceExcludedWithRegex(resourceName string, excludeAll bool, excludeThese []string, excludeTheseRegexes []string, excludeAllButThese []string) bool {
	if strings.TrimSpace(resourceName) == "" {
		return true
	}

	if excludeAll {
		return true
	}

	if excludeThese != nil {
		// Ignore any empty strings
		filteredList := lo.Filter(excludeThese, func(item string, index int) bool {
			return strings.TrimSpace(item) != ""
		})

		if len(filteredList) != 0 && slices.Index(filteredList, resourceName) != -1 {
			return true
		}
	}

	if excludeAllButThese != nil && len(excludeAllButThese) != 0 {
		// Ignore any empty strings
		filteredList := lo.Filter(excludeAllButThese, func(item string, index int) bool {
			return strings.TrimSpace(item) != ""
		})

		if len(filteredList) != 0 && slices.Index(filteredList, resourceName) == -1 {
			return true
		}
	}

	if excludeTheseRegexes != nil {
		matched := lo.ContainsBy(excludeTheseRegexes, func(item string) bool {
			if strings.TrimSpace(item) == "" {
				return false
			}

			r, err := regexp.Compile(item)

			if err != nil {
				return false
			}

			return r.MatchString(resourceName)
		})

		if matched {
			return true
		}
	}

	return false
}

func (e DefaultExcluder) FilteredTenantTags(tenantTags []string, excludeTenantTags args.StringSliceArgs, excludeTenantTagSets args.StringSliceArgs) []string {
	if tenantTags == nil {
		return []string{}
	}

	tags := lo.Filter(tenantTags, func(item string, index int) bool {
		if e.IsResourceExcluded(item, false, excludeTenantTags, nil) {
			return false
		}

		split := strings.Split(item, "/")

		// Exclude the tag if it is part of an excluded tag set
		return !e.IsResourceExcluded(split[0], false, excludeTenantTagSets, nil)
	})

	return tags
}
