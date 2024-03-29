package converters

import "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"

// ExcludeByName has logic for excluding resources based on some filters. Excluded resources are typically supplied
// from the command line.
type ExcludeByName interface {
	IsResourceExcluded(resourceName string, excludeAll bool, excludeThese []string, excludeAllButThese []string) bool
	IsResourceExcludedWithRegex(resourceName string, excludeAll bool, excludeThese []string, excludeTheseRegexes []string, excludeAllButThese []string) bool
	FilteredTenantTags(tenantTags []string, excludeTenantTags args.StringSliceArgs, excludeTenantTagSets args.StringSliceArgs) []string
}
