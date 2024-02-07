package data

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"go.uber.org/zap"
)

type ToHcl func() (string, error)

type ResourceParameter struct {
	Name        string
	Description string
	Label       string
	Type        string
	Sensitive   bool
}

// ResourceDetails is used to capture the dependencies required by the root resources that was
// exported. The process works like this:
// 1. The root resources is captured in a ResourceDetails from the Octopus API.
// 2. Any dependencies of the root object are captured in their own ResourceDetails objects.
// 3. Repeat step 2 for dependencies of dependencies.
// 4. Once all dependencies are captured, run ToHcl feeding in the collection of ResourceDetails built in steps 1 - 3.
// 5. ToHcl converts the object to HCL, and uses the Lookup field in the appropriate ResourceDetails to reference a dependency.
type ResourceDetails struct {
	// Id is the octopus ID of the exported resource
	Id string
	// ResourceType is the type of Octopus resource (almost always related to the path that the resource is loaded from)
	ResourceType string
	// Lookup is the ID of the resource created or looked up by Terraform
	Lookup string
	// Dependency provides a way for one resource to depend on this resource. Usually the same of the Lookup, but can be
	// a reference to a group of resources in stateless mode.
	Dependency string
	// FileName is the file contains the exported resource
	FileName string
	// ToHCL is a function that generates the HCL from the Octopus resource
	ToHcl ToHcl
	// A collection of any parameters that relate to the resource
	Parameters []ResourceParameter
}

type ResourceDetailsCollection struct {
	Resources []ResourceDetails
}

// HasResource returns true if the resource with the id and resourceType exist in the collection, and false otherwise
func (c *ResourceDetailsCollection) HasResource(id string, resourceType string) bool {
	for _, r := range c.Resources {
		if r.Id == id && r.ResourceType == resourceType {
			return true
		}
	}

	return false
}

// AddResource adds a resource to the collection
func (c *ResourceDetailsCollection) AddResource(resource ...ResourceDetails) {
	if c.Resources == nil {
		c.Resources = []ResourceDetails{}
	}

	c.Resources = append(c.Resources, resource...)
}

// GetAllResource returns a slice of resources in the collection of type resourceType
func (c *ResourceDetailsCollection) GetAllResource(resourceType string) []ResourceDetails {
	resources := make([]ResourceDetails, 0)
	for _, r := range c.Resources {
		if r.ResourceType == resourceType {
			resources = append(resources, r)
		}
	}

	return resources
}

// GetResource returns the terraform references for a given resource type and id.
// If the resource is not found, an empty string is returned. There is no valid reason to return an empty string,
// but we treat a mostly valid output as a "graceful fallback" rather than failing hard, as the resulting text
// can still be edited by hand.
func (c *ResourceDetailsCollection) GetResource(resourceType string, id string) string {
	for _, r := range c.Resources {
		if r.Id == id && r.ResourceType == resourceType {
			return r.Lookup
		}
	}

	zap.L().Error("Failed to resolve lookup " + id + " of type " + resourceType)

	return ""
}

// GetResourceDependency returns the terraform references for a given resource type and id.
// The returned string is used only for the depends_on field, as it may reference to a collection of resources
// rather than a single ID.
func (c *ResourceDetailsCollection) GetResourceDependency(resourceType string, id string) string {
	for _, r := range c.Resources {
		if r.Id == id && r.ResourceType == resourceType {
			// return the dependency field if it was defined, otherwise fall back to the lookup field
			return strutil.DefaultIfEmpty(r.Dependency, r.Lookup)
		}
	}

	zap.L().Error("Failed to resolve dependency " + id + " of type " + resourceType)

	return ""
}

// GetResources returns the Terraform references for resources of the given type and with the supplied ids.
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

// GetResourcePointer returns the Terraform reference for a given resource type and id as a string pointer.
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
