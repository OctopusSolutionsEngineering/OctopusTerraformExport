package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"go.uber.org/zap"
)

type ToHcl func() (string, error)

// ResourceDetails is used to capture the dependencies required by the root resources that was
// exported. The process works like this:
// 1. The root resources is captured from the Octopus API.
// 2. Any dependencies are captured in a ResourceDetails object.
// 3. Repeat step 2 for dependencies of dependencies.
// 4. Once all dependencies are captured, run ToHcl feeding in the collection of ResourceDetails built in steps 1 - 3.
// 5. ToHcl converts the object to HCL, and uses the Lookup field in the appropriate ResourceDetails to reference a dependency.
type ResourceDetails struct {
	Id           string
	ResourceType string
	Lookup       string
	FileName     string
	ToHcl        ToHcl
}

type ResourceDetailsCollection struct {
	Resources []ResourceDetails
}

func (c *ResourceDetailsCollection) HasResource(id string, resourceType string) bool {
	for _, r := range c.Resources {
		if r.Id == id && r.ResourceType == resourceType {
			return true
		}
	}

	return false
}

func (c *ResourceDetailsCollection) AddResource(resource ...ResourceDetails) {
	if c.Resources == nil {
		c.Resources = []ResourceDetails{}
	}

	c.Resources = append(c.Resources, resource...)
}

func (c *ResourceDetailsCollection) GetAllResource(resourceType string) []ResourceDetails {
	resources := make([]ResourceDetails, 0)
	for _, r := range c.Resources {
		if r.ResourceType == resourceType {
			resources = append(resources, r)
		}
	}

	return resources
}

func (c *ResourceDetailsCollection) GetResource(resourceType string, id string) string {
	for _, r := range c.Resources {
		if r.Id == id && r.ResourceType == resourceType {
			return r.Lookup
		}
	}

	zap.L().Error("Failed to resolve " + id + " of type " + resourceType)

	return ""
}

func (c *ResourceDetailsCollection) GetResources(resourceType string, ids ...string) []string {
	lookups := []string{}
	for _, r := range c.Resources {
		for _, i := range ids {
			if r.Id == i && r.ResourceType == resourceType {
				lookups = append(lookups, r.Lookup)
			}
		}
	}

	return lookups
}

func (c *ResourceDetailsCollection) GetResourcePointer(resourceType string, id *string) *string {
	if id != nil {
		for _, r := range c.Resources {
			if r.Id == *id && r.ResourceType == resourceType {
				return &r.Lookup
			}
		}

		zap.L().Error("Failed to resolve " + strutil.EmptyIfNil(id) + " of type " + resourceType)
	}

	empty := ""
	return &empty
}
