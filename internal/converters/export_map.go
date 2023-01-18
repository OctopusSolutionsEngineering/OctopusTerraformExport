package converters

type ToHcl func(map[string]ResourceDetails) (string, error)

// ResourceDetails captures the ID and resource type of an Octopus API resource, the HCL lookup
// that dependant objects can use to reference the imported object, and a function used\
// to generate the HCL.
type ResourceDetails struct {
	Id           string
	ResourceType string
	Lookup       string
	FileName     string
	ToHcl        ToHcl
}
