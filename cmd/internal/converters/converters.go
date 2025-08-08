package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
)

// ConverterById converts an individual resource by its ID
type ConverterById interface {
	// ToHclById converts a single resource by its ID. This is used when converting a single project,
	// and then converting anything that the project references (like feeds, accounts, environments etc).
	ToHclById(id string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterToStatelessById converts an individual resource by its ID
type ConverterToStatelessById interface {
	// ToHclStatelessById converts a single resource to a stateless representation by its ID. This is used when
	// converting a single project, and then converting anything that the project references (like feeds, accounts,
	// environments etc).
	ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterByIdWithLookups converts an individual resource by its ID, with all external resources referenced
// as lookups
type ConverterByIdWithLookups interface {
	// ToHclByIdWithLookups converts a single resource by its ID. This is used when converting a single project,
	// and then referencing all other resources by lookup (like feeds, accounts, environments etc).
	ToHclByIdWithLookups(id string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterLookupById converts an individual resource by its ID to a data lookup
type ConverterLookupById interface {
	// ToHclLookupById is used to create a data resource that queries a space for an existing resource. This
	// is used when converting a project with the -lookupProjectDependencies argument specified. It allows a project
	// to reference existing resources like accounts, feeds, environments etc in the space in which the project
	// is imported.
	ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterAndLookupById converts an individual resource by ID to HCL and to a data lookup
type ConverterAndLookupById interface {
	ConverterById
	ConverterLookupById
}

// ConverterWithStatelessById converts an individual resource by ID to HCL, either stateless or normal
type ConverterWithStatelessById interface {
	ConverterById
	ConverterToStatelessById
}

// ConverterAndLookupWithStatelessById converts an individual resource by ID to HCL and to a data lookup
type ConverterAndLookupWithStatelessById interface {
	ConverterById
	ConverterLookupById
	ConverterToStatelessById
}

// ConverterAndLookupWithStatelessByIdAndSystemData converts an individual resource by ID to HCL and to a data lookup
// It also exports system data, which are data sources that are used in every space regardless of the resources in the space.
type ConverterAndLookupWithStatelessByIdAndSystemData interface {
	ConverterById
	ConverterLookupById
	ConverterToStatelessById
	SystemDataToHcl(dependencies *data.ResourceDetailsCollection)
}

// ConverterAndWithLookupsById converts an individual resource by ID to HCL, either exporting all dependencies,
// or looking up all dependencies
type ConverterAndWithLookupsById interface {
	ConverterById
	ConverterLookupById
	ConverterByIdWithLookups
	ConverterToStatelessById
}

// ConverterByIdWithName converts an individual resource by its ID, and uses the supplied name for the Terraform resource
type ConverterByIdWithName interface {
	ToHclByIdAndName(id string, name string, recursive bool, dependencies *data.ResourceDetailsCollection) error
	ToHclStatelessByIdAndName(id string, name string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterLookupByIdWithName converts an individual resource by its ID, uses the supplied name for the Terraform resource, and
// references external resources via a data source lookup
type ConverterLookupByIdWithName interface {
	ToHclLookupByIdAndName(id string, name string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterLookupByIdWithBranch converts an individual resource by its git branch, and references external resources via a data source lookup
type ConverterLookupByIdWithBranch interface {
	ToHclLookupByIdAndBranch(parentId string, branch string, dependencies *data.ResourceDetailsCollection) error
	ToHclByIdAndBranch(parentId string, branch string, recursive bool, dependencies *data.ResourceDetailsCollection) error
	ToHclStatelessByIdAndBranch(parentId string, branch string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterLookupByIdWithBranchAndProject converts an individual resource by its git branch based on a parent project,
// and references external resources via a data source lookup.
// This interface is needed when looking up resources that can only be found from a parent project, such as CaC runbooks.
type ConverterLookupByIdWithBranchAndProject interface {
	ToHclLookupByIdBranchAndProject(projectId string, resourceId string, branch string, dependencies *data.ResourceDetailsCollection) error
	ToHclByIdBranchAndProject(projectId string, resourceId string, branch string, recursive bool, dependencies *data.ResourceDetailsCollection) error
	ToHclStatelessByIdBranchAndProject(projectId string, resourceId string, branch string, dependencies *data.ResourceDetailsCollection) error
}

type ActionProcessor interface {
	SetActionProcessor(actionProcessor *OctopusActionProcessor)
}

// ConverterAndLookupByIdAndName converts an individual resource by ID to HCL and to a data lookup
type ConverterAndLookupByIdAndName interface {
	ConverterByIdWithName
	ConverterLookupByIdWithName
}

// ConverterAndLookupByIdAndNameOrBranch converts an individual resource by ID or git branch to HCL and to a data lookup
type ConverterAndLookupByIdAndNameOrBranch interface {
	ConverterByIdWithName
	ConverterLookupByIdWithName
	ConverterLookupByIdWithBranch
}

// ConverterAndLookupByIdAndNameWithDeploymentProcesses converts an individual resource by ID to HCL and to a data lookup
// with references to projects
type ConverterAndLookupByIdAndNameWithDeploymentProcesses interface {
	ConverterById
	ConverterLookupById
	ConverterToStatelessById
	ActionProcessor
}

// ConverterAndLookupByIdAndNameOrBranchWithDeploymentProcesses converts an individual resource by ID or git branch to HCL and to a data lookup
// with references to projects
type ConverterAndLookupByIdAndNameOrBranchWithDeploymentProcesses interface {
	ConverterAndLookupByIdAndNameWithDeploymentProcesses
	ConverterLookupByIdWithBranch
	ActionProcessor
}

// ConverterAndLookupByIdAndNameOrBranchAndProjectWithDeploymentProcesses converts an individual resource by ID or git branch to HCL and based on a parent project
type ConverterAndLookupByIdAndNameOrBranchAndProjectWithDeploymentProcesses interface {
	ConverterAndLookupByIdAndNameWithDeploymentProcesses
	ConverterLookupByIdWithBranchAndProject
	ActionProcessor
}

// ConverterByIdWithNameAndParent converts a resource by its ID, uses the supplied name, and has a reference to its parent
type ConverterByIdWithNameAndParent interface {
	// ToHclByIdAndName converts a resource with a reference to a parent object
	// id is the ID of the resource to convert.
	// recursive indicates if the dependencies of this resource should also be exported.
	// name is the name of the resource to convert.
	// parentLookup is an HCL interpolation that resolves the parents ID.
	// parentCount is an HCL interpolation that resolves to 1 or 0 based on whether the parent is created or not in a stateless module.
	// dependencies is the collection of exported resources.
	ToHclByIdAndName(id string, recursive bool, name string, parentLookup string, parentCount *string, dependencies *data.ResourceDetailsCollection) error
	// ToHclStatelessByIdAndName converts a resource with a reference to a parent object
	// id is the ID of the resource to convert.
	// recursive indicates if the dependencies of this resource should also be exported.
	// name is the name of the resource to convert.
	// parentLookup is an HCL interpolation that resolves the parents ID.
	// parentCount is an HCL interpolation that resolves to 1 or 0 based on whether the parent is created or not in a stateless module.
	// dependencies is the collection of exported resources.
	ToHclStatelessByIdAndName(id string, recursive bool, name string, parentLookup string, parentCount *string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterLookupByIdWithNameAndParent converts a resource by its ID, uses the supplied name, and has a reference to its parent, and
// references external resources via data source lookups
type ConverterLookupByIdWithNameAndParent interface {
	ToHclLookupByIdAndName(id string, name string, parentLookup string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterAndLookupByIdWithNameAndParent converts a resource by its ID, uses the supplied name, and has a reference to its parent, and
// can also reference external resources via data source lookups
type ConverterAndLookupByIdWithNameAndParent interface {
	ConverterByIdWithNameAndParent
	ConverterLookupByIdWithNameAndParent
}

// ConverterByProjectIdWithName converts objects based on their relationship to a project, and uses the supplied name for the Terraform resource
type ConverterByProjectIdWithName interface {
	ToHclByProjectIdAndName(id string, name string, recursive bool, lookup bool, dependencies *data.ResourceDetailsCollection) error
	ToHclStatelessByProjectIdAndName(id string, name string, recursive bool, lookup bool, dependencies *data.ResourceDetailsCollection) error
}

// ConverterByProjectId converts objects based on their relationship to a project
type ConverterByProjectId interface {
	ToHclByProjectId(projectId string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterToStatelessByProjectId converts objects to stateless representations based on their relationship to a project
type ConverterToStatelessByProjectId interface {
	ToHclStatelessByProjectId(projectId string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterLookupByProjectId converts objects to data lookups based on their relationship to a project
type ConverterLookupByProjectId interface {
	ToHclLookupByProjectId(projectId string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterAndLookupByProjectId converts objects to HCL and data lookups based on their relationship to a project
type ConverterAndLookupByProjectId interface {
	ConverterByProjectId
	ConverterToStatelessByProjectId
	ConverterLookupByProjectId
}

// ConverterByProjectIdAndName converts objects based on their relationship to a project, with the ability to reference the parent
type ConverterByProjectIdAndName interface {
	ToHclByProjectIdAndName(projectId string, parentName string, parentLookup string, parentCount *string, recursive bool, dependencies *data.ResourceDetailsCollection) error
}

// ConverterLookupByProjectIdAndName converts objects to data lookups based on their relationship to a project branch, with the ability to reference the parent
type ConverterLookupByProjectIdAndName interface {
	ToHclLookupByProjectIdAndName(projectId string, parentName string, parentLookup string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterByProjectIdBranchAndName converts objects based on their relationship to a project branch, with the ability to reference the parent
type ConverterByProjectIdBranchAndName interface {
	ToHclByProjectIdBranchAndName(projectId string, branch string, parentName string, parentLookup string, parentCount *string, recursive bool, dependencies *data.ResourceDetailsCollection) error
	ToHclStatelessByProjectIdBranchAndName(projectId string, branch string, parentName string, parentLookup string, parentCount *string, recursive bool, dependencies *data.ResourceDetailsCollection) error
}

// ConverterLookupByProjectIdBranchAndName converts objects to data lookups based on their relationship to a project, with the ability to reference the parent
type ConverterLookupByProjectIdBranchAndName interface {
	ToHclLookupByProjectIdBranchAndName(projectId string, branch string, parentName string, parentLookup string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterAndLookupByProjectIdAndName converts objects to HCL and data lookups based on their relationship to a project
type ConverterAndLookupByProjectIdAndName interface {
	ConverterByProjectIdAndName
	ConverterLookupByProjectIdAndName
	ConverterByProjectIdBranchAndName
	ConverterLookupByProjectIdBranchAndName
}

// ConverterByProjectIdWithTerraDependencies converts objects based on their relationship to a project, with manual terraform dependencies
type ConverterByProjectIdWithTerraDependencies interface {
	ToHclByProjectIdWithTerraDependencies(projectId string, terraformDependencies map[string]string, dependencies *data.ResourceDetailsCollection) error
	ToHclStatelessByProjectIdWithTerraDependencies(projectId string, terraformDependencies map[string]string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterLookupByProjectIdWithTerraDependencies converts objects based on their relationship to a project, with manual terraform dependencies, and using a lookup for dependencies
type ConverterLookupByProjectIdWithTerraDependencies interface {
	ToHclLookupByProjectIdWithTerraDependencies(projectId string, terraformDependencies map[string]string, dependencies *data.ResourceDetailsCollection) error
}

// ConverterAndLookupByProjectIdWithTerraDependencies converts objects to HCL and data lookups based on their relationship to a project
type ConverterAndLookupByProjectIdWithTerraDependencies interface {
	ConverterByProjectIdWithTerraDependencies
	ConverterLookupByProjectIdWithTerraDependencies
}

// ConverterByTenantId converts objects based on the relationship to a tenant
type ConverterByTenantId interface {
	ToHclByTenantId(projectId string, stateless bool, dependencies *data.ResourceDetailsCollection) error
}

// ConvertToHclByResource converts objects directly
type ConvertToHclByResource[C any] interface {
	ToHclByResource(resource C, dependencies *data.ResourceDetailsCollection) error
	ToHclByResourceStateless(resource C, dependencies *data.ResourceDetailsCollection) error
}

// ConvertToHclLookupByResource creates a data lookup from the objects
type ConvertToHclLookupByResource[C any] interface {
	ToHclLookupByResource(resource C, dependencies *data.ResourceDetailsCollection) error
}

// Converter converts all objects in bulk
type Converter interface {
	// AllToHcl converts all the resources of a given type to HCL. This is used when converting a space.
	AllToHcl(dependencies *data.ResourceDetailsCollection)
	// AllToStatelessHcl converts all the resources of a given type to a stateless HCL module suitable for a step template.
	AllToStatelessHcl(dependencies *data.ResourceDetailsCollection)
}

// Converter converts all objects in bulk, and exports system data
type ConverterWithSystemData interface {
	Converter
	SystemDataToHcl(dependencies *data.ResourceDetailsCollection)
}

// ToHclByTenantIdAndProject exports tenant common and project variables for a project.
// This is used when the project is responsible for defining the variables associated with an existing tenant when the
// tenant could not define any variables until the project was available.
type ToHclByTenantIdAndProject interface {
	ToHclByTenantIdAndProject(id string, project octopus.Project, dependencies *data.ResourceDetailsCollection) error
}
