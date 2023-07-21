package converters

// ExcludeByName has logic for excluding resources based on some filters
type ExcludeByName interface {
	IsResourceExcluded(resourceName string, excludeAll bool, excludeThese []string, excludeAllButThese []string) bool
}
