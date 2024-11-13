package comparable

// OctopusResource is an interface that defines the methods that all Octopus resources must implement
// in order to be compared to another similar resource.
type OctopusResource interface {
	// Compare compares the current resource to another resource and returns a comparison object
	Compare(other OctopusResource) OctopusResourceComparison
	// GetChildResources returns a slice of child resources of the current resource
	GetChildResources() []OctopusResource
	// GetName returns the name of the resource
	GetName() string
}

// OctopusResourceComparison is a struct that defines the comparison between two Octopus resources
type OctopusResourceComparison struct {
	// SourceResource is the source resource that was compared (i.e this resource)
	SourceResource OctopusResource
	// DestinationResource is the destination resource that was compared (i.e the other resource)
	DestinationResource OctopusResource
	// Differences is a map of the differences between the source and destination resources
	Differences map[string]Differences
	// ChildOctopusResourceComparison is a map of the child resources of the source resource and their comparisons
	ChildOctopusResourceComparison map[string]OctopusResourceComparison
}

type Differences struct {
	SourceValue      string
	DestinationValue string
}
