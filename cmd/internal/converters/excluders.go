package converters

// ExcludeByName has logic for excluding resources based on some filters. Excluded resources are typically supplied
// from the command line.
type ExcludeByName interface {
	IsResourceExcluded(resourceName string, excludeAll bool, excludeThese []string, excludeAllButThese []string) bool
}
