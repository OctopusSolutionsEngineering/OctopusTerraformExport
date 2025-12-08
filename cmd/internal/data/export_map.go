package data

import (
	"sync"

	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type ToHcl func() (string, error)

type ResourceParameter struct {
	VariableName  string
	Description   string
	Label         string
	ResourceName  string
	ParameterType string
	Sensitive     bool
	DefaultValue  string
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
	// AlternateId is the alternate octopus ID of the exported resource.
	// This can occur when a single Terraform resource represents multiple Octopus resources.
	// An example is an octopusdeploy_process_step resource, which represents the combination of a step
	// and the first action in the step. Sometimes we need to reference the step by its ID, and sometimes
	// we need to reference the action by its ID.
	AlternateId string
	// ParentId is an optional field that allows a resource to define its parent.
	// This is useful when establishing dependencies between Terraform resources where it is not easy to identify the
	// individual Terraform resources that belong to a parent. For example, a channel must depend on the steps in a project
	// because a channel references step packages by name, and thus do not establish a direct relationship that can be
	// deduced by Terraform. However, it is not easy to infer all the step resources that belong to a project based on ID
	// alone. But by setting the ParentId field, it is possible to query all the steps that belong to a project.
	ParentId string
	// Name is the name of the resource
	Name string
	// Step templates have a calculated version value that is only available when the template is created. This value
	// is an expression that is used to reference the newly created version value.
	VersionLookup string
	// Step templates have a version field that is calculated by Octopus. This value captures the current value from
	// the space being exported.
	VersionCurrent string
	// Step templates can be based on community step templates. The URL of the community step template is used as
	// and external ID that links resources between spaces.
	ExternalID string
	// Some resources are sorted in a specific order and need to be recreated in the same order.
	// Environments are an example of this.
	SortOrder int
	// ResourceType is the type of Octopus resource (almost always related to the path that the resource is loaded from)
	ResourceType string
	// Lookup is the ID of the resource created or looked up by Terraform. For example,
	// "${octopusdeploy_project.my_project.id}".
	// Lookup is the ID of the resource, while Dependency is the name of the resource in the Terraform.
	Lookup string
	// Dependency provides a way for one resource to depend on this resource. It is a reference to the terraform
	// resource, for example, "${octopusdeploy_project.my_project}".
	Dependency string
	// Count stores the HCL assigned to the count attribute. This is useful when child resources need to have the same
	// count value as a parent.
	Count string
	// FileName is the file contains the exported resource
	FileName string
	// ToHCL is a function that generates the HCL from the Octopus resource
	ToHcl ToHcl
	// A collection of any parameters that relate to the resource. These are used when building up a step template.
	Parameters []ResourceParameter
}

// The DummyVariableReference struct defines the details of a variable that had a dummy value injected into it.
type DummyVariableReference struct {
	VariableName string
	ResourceName string
	ResourceType string
}

type ResourceDetailsCollection struct {
	Resources      []ResourceDetails
	DummyVariables []DummyVariableReference
	// A mutex to protect lookups
	mu sync.Mutex
}

// AddDummy adds a dummy variable reference to the collection
func (c *ResourceDetailsCollection) AddDummy(reference DummyVariableReference) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.DummyVariables == nil {
		c.DummyVariables = []DummyVariableReference{}
	}

	c.DummyVariables = append(c.DummyVariables, reference)
}

/*
HasResource returns true if the resource with the id and resourceType exist in the collection, and false otherwise.
While this method is thread-safe, it is not a guarantee that two goroutines are not processing the same resource
concurrently. If HasResource returns true, it is safe to assume the resource has been processed by other goroutines
and exit early. If HasResource returns false, the resource should be processed, but the results may be discarded
by the AddResource method if another goroutine has processed the same resource in the meantime.
*/
func (c *ResourceDetailsCollection) HasResource(id string, resourceType string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, r := range c.Resources {
		if r.Id == id && r.ResourceType == resourceType {
			return true
		}
	}

	return false
}

func (c *ResourceDetailsCollection) AddResourcePtr(resources ...*ResourceDetails) {
	for _, resource := range resources {
		if resource != nil {
			c.AddResource(*resource)
		}
	}
}

// AddResource adds a resource to the collection
func (c *ResourceDetailsCollection) AddResource(resources ...ResourceDetails) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Resources == nil {
		c.Resources = []ResourceDetails{}
	}

	if resources == nil {
		return
	}

	/*
		When running with multiple goroutines it is possible to have a race condition where a call to HasResource
		returns false, indicating that a converter should go ahead and process the resource. But by the time
		AddResource is called, another goroutine has added the same resource. This check is to ensure that the
		resource is not added twice.
	*/
	fixedResources := lo.Filter(resources, func(resource ResourceDetails, index int) bool {
		return !lo.ContainsBy(c.Resources, func(existingResource ResourceDetails) bool {
			return resource.Id != "" && resource.ResourceType != "" && existingResource.Id == resource.Id && existingResource.ResourceType == resource.ResourceType
		})
	})

	c.Resources = append(c.Resources, fixedResources...)
}

// GetAllResource returns a slice of resources in the collection of type resourceType
func (c *ResourceDetailsCollection) GetAllResource(resourceType string) []ResourceDetails {
	c.mu.Lock()
	defer c.mu.Unlock()

	resources := make([]ResourceDetails, 0)
	for _, r := range c.Resources {
		if r.ResourceType == resourceType {
			resources = append(resources, r)
		}
	}

	return resources
}

// GetAllResourceWithLowerSort returns a slice of resources in the collection of type resourceType that have
// a lower sort order.
func (c *ResourceDetailsCollection) GetAllResourceWithLowerSort(resourceType string, maxSort int) []ResourceDetails {
	return lo.Filter(c.GetAllResource(resourceType), func(item ResourceDetails, index int) bool {
		return item.SortOrder < maxSort
	})
}

// GetResource returns the terraform references for a given resource type and id.
// If the resource is not found, an empty string is returned. There is no valid reason to return an empty string,
// but we treat a mostly valid output as a "graceful fallback" rather than failing hard, as the resulting text
// can still be edited by hand.
func (c *ResourceDetailsCollection) GetResource(resourceType string, id string) string {
	if id == "" {
		return ""
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, r := range c.Resources {
		if (r.Id == id || r.AlternateId == id) && r.ResourceType == resourceType {
			return r.Lookup
		}
	}

	zap.L().Error("Failed to resolve lookup " + id + " of type " + resourceType)

	return ""
}

// GetResourceCount returns the terraform count attribute for a given resource type and id.
// The returned string is used only for the depends_on field, as it may reference to a collection of resources
// rather than a single ID.
func (c *ResourceDetailsCollection) GetResourceCount(resourceType string, id string) string {
	if id == "" {
		return ""
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, r := range c.Resources {
		if r.Id == id && r.ResourceType == resourceType {
			return r.Count
		}
	}

	zap.L().Error("Failed to resolve lookup " + id + " of type " + resourceType)

	return ""
}

// GetResourceName returns the terraform name attribute for a given resource type and id.
// The returned string is used only for the depends_on field, as it may reference to a collection of resources
// rather than a single ID.
func (c *ResourceDetailsCollection) GetResourceName(resourceType string, id string) string {
	if id == "" {
		return ""
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, r := range c.Resources {
		if r.Id == id && r.ResourceType == resourceType {
			return r.Name
		}
	}

	zap.L().Error("Failed to resolve lookup " + id + " of type " + resourceType)

	return ""
}

// GetResourceVersionLookup returns the terraform syntax to lookup the version of the new resource.
func (c *ResourceDetailsCollection) GetResourceVersionLookup(resourceType string, id string) string {
	if id == "" {
		return ""
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, r := range c.Resources {
		if r.Id == id && r.ResourceType == resourceType {
			return r.VersionLookup
		}
	}

	zap.L().Error("Failed to resolve lookup " + id + " of type " + resourceType)

	return ""
}

// GetResourceVersionCurrent returns the current version of the resource being exported.
func (c *ResourceDetailsCollection) GetResourceVersionCurrent(resourceType string, id string) string {
	if id == "" {
		return ""
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, r := range c.Resources {
		if r.Id == id && r.ResourceType == resourceType {
			return r.VersionCurrent
		}
	}

	zap.L().Error("Failed to resolve lookup " + id + " of type " + resourceType)

	return ""
}

// GetResourceDependency returns the terraform references for a given resource type and id.
// The returned string is used only for the depends_on field, as it may reference to a collection of resources
// rather than a single ID.
func (c *ResourceDetailsCollection) GetResourceDependency(resourceType string, id string) string {
	if id == "" {
		return ""
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, r := range c.Resources {
		if (r.Id == id || r.AlternateId == id) && r.ResourceType == resourceType {
			// return the dependency field if it was defined, otherwise fall back to the lookup field
			return strutil.DefaultIfEmpty(r.Dependency, r.Lookup)
		}
	}

	zap.L().Error("Failed to resolve dependency " + id + " of type " + resourceType)

	return ""
}

// GetResourceDependencyPointer returns the terraform references for a given resource type and id.
// The returned string is used only for the depends_on field, as it may reference to a collection of resources
// rather than a single ID.
func (c *ResourceDetailsCollection) GetResourceDependencyPointer(resourceType string, id *string) *string {
	if id == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, r := range c.Resources {
		if r.Id == *id && r.ResourceType == resourceType {
			// return the dependency field if it was defined, otherwise fall back to the lookup field
			return strutil.NilIfEmpty(strutil.DefaultIfEmpty(r.Dependency, r.Lookup))
		}
	}

	zap.L().Error("Failed to resolve dependency " + *id + " of type " + resourceType)

	return nil
}

// GetResourceDependencyFromParent returns the terraform references for a given resource type based on the parent ID.
func (c *ResourceDetailsCollection) GetResourceDependencyFromParent(parentId string, resourceType string) []string {
	if parentId == "" {
		return []string{}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return lo.FilterMap(c.Resources, func(item ResourceDetails, index int) (string, bool) {
		return item.Dependency, item.ParentId == parentId && item.ResourceType == resourceType
	})
}

// GetResources returns the Terraform references for resources of the given type and with the supplied ids.
func (c *ResourceDetailsCollection) GetResources(resourceType string, ids ...string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	lookups := []string{}
	for _, i := range ids {
		found := false
		for _, r := range c.Resources {
			if r.Id == i && r.ResourceType == resourceType {
				lookups = append(lookups, r.Lookup)
				found = true
				continue
			}
		}
		if !found {
			zap.L().Error("Failed to resolve " + i + " of type " + resourceType)
		}
	}

	return lookups
}

// GetResourcePointer returns the Terraform reference for a given resource type and id as a string pointer.
func (c *ResourceDetailsCollection) GetResourcePointer(resourceType string, id *string) *string {
	if id == nil {
		empty := ""
		return &empty
	}

	c.mu.Lock()
	defer c.mu.Unlock()

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
