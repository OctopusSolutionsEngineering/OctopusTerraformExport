package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/samber/lo"
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

func (e DefaultExcluder) FilteredTenantTags(tenantTags []string, excludeTenantTags args.ExcludeTenantTags, excludeTenantTagSets args.ExcludeTenantTagSets) []string {
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
