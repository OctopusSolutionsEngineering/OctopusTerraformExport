package converters

// DummySecretGenerator defines the service used to generate dummy secret values
type DummySecretGenerator interface {
	GetDummySecret() *string
}

// ConverterById converts an individual resource by its ID
type ConverterById interface {
	// ToHclById converts a single resource by its ID. This is used when converting a single project,
	// and then converting anything that the project references (like feeds, accounts, environments etc).
	ToHclById(id string, dependencies *ResourceDetailsCollection) error
}

// ConverterLookupById converts an individual resource by its ID to a data lookup
type ConverterLookupById interface {
	// ToHclLookupById is used to create a data resource that queries a space for an existing resource. This
	// is used when converting a project with the -lookupProjectDependencies argument specified. It allows a project
	// to reference existing resources like accounts, feeds, environments etc in the space in which the project
	// is imported.
	ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error
}

// ConverterAndLookupById converts an individual resource by ID to HCL and to a data lookup
type ConverterAndLookupById interface {
	ConverterById
	ConverterLookupById
}

// ConverterByIdWithName converts an individual resource by its ID, and uses the supplied name for the Terraform resource
type ConverterByIdWithName interface {
	ToHclByIdAndName(id string, name string, dependencies *ResourceDetailsCollection) error
}

// ConverterLookupByIdWithName converts an individual resource by its ID, uses the supplied name for the Terraform resource, and
// references external resources via a data source lookup
type ConverterLookupByIdWithName interface {
	ToHclLookupByIdAndName(id string, name string, dependencies *ResourceDetailsCollection) error
}

// ConverterAndLookupByIdAndName converts an individual resource by ID to HCL and to a data lookup
type ConverterAndLookupByIdAndName interface {
	ConverterByIdWithName
	ConverterLookupByIdWithName
}

// ConverterByIdWithNameAndParent converts a resource by its ID, uses the supplied name, and has a reference to its parent
type ConverterByIdWithNameAndParent interface {
	ToHclByIdAndName(id string, name string, parentLookup string, dependencies *ResourceDetailsCollection) error
}

// ConverterLookupByIdWithNameAndParent converts a resource by its ID, uses the supplied name, and has a reference to its parent, and
// references external resources via data source lookups
type ConverterLookupByIdWithNameAndParent interface {
	ToHclLookupByIdAndName(id string, name string, parentLookup string, dependencies *ResourceDetailsCollection) error
}

// ConverterAndLookupByIdWithNameAndParent converts a resource by its ID, uses the supplied name, and has a reference to its parent, and
// can also reference external resources via data source lookups
type ConverterAndLookupByIdWithNameAndParent interface {
	ConverterByIdWithNameAndParent
	ConverterLookupByIdWithNameAndParent
}

// ConverterByProjectIdWithName converts objects based on their relationship to a project, and uses the supplied name for the Terraform resource
type ConverterByProjectIdWithName interface {
	ToHclByProjectIdAndName(id string, name string, dependencies *ResourceDetailsCollection) error
}

// ConverterByProjectId converts objects based on their relationship to a project
type ConverterByProjectId interface {
	ToHclByProjectId(projectId string, dependencies *ResourceDetailsCollection) error
}

// ConverterLookupByProjectId converts objects to data lookups based on their relationship to a project
type ConverterLookupByProjectId interface {
	ToHclLookupByProjectId(projectId string, dependencies *ResourceDetailsCollection) error
}

// ConverterAndLookupByProjectId converts objects to HCL and data lookups based on their relationship to a project
type ConverterAndLookupByProjectId interface {
	ConverterByProjectId
	ConverterLookupByProjectId
}

// ConverterByProjectIdAndName converts objects based on their relationship to a project, with the ability to reference the parent
type ConverterByProjectIdAndName interface {
	ToHclByProjectIdAndName(projectId string, parentName string, parentLookup string, dependencies *ResourceDetailsCollection) error
}

// ConverterLookupByProjectIdAndName converts objects to data lookups based on their relationship to a project, with the ability to reference the parent
type ConverterLookupByProjectIdAndName interface {
	ToHclLookupByProjectIdAndName(projectId string, parentName string, parentLookup string, dependencies *ResourceDetailsCollection) error
}

// ConverterAndLookupByProjectIdAndName converts objects to HCL and data lookups based on their relationship to a project
type ConverterAndLookupByProjectIdAndName interface {
	ConverterByProjectIdAndName
	ConverterLookupByProjectIdAndName
}

// ConverterByProjectIdWithTerraDependencies converts objects based on their relationship to a project, with manual terraform dependencies
type ConverterByProjectIdWithTerraDependencies interface {
	ToHclByProjectIdWithTerraDependencies(projectId string, terraformDependencies map[string]string, dependencies *ResourceDetailsCollection) error
}

// ConverterLookupByProjectIdWithTerraDependencies converts objects based on their relationship to a project, with manual terraform dependencies, and using a lookup for dependencies
type ConverterLookupByProjectIdWithTerraDependencies interface {
	ToHclLookupByProjectIdWithTerraDependencies(projectId string, terraformDependencies map[string]string, dependencies *ResourceDetailsCollection) error
}

// ConverterAndLookupByProjectIdWithTerraDependencies converts objects to HCL and data lookups based on their relationship to a project
type ConverterAndLookupByProjectIdWithTerraDependencies interface {
	ConverterByProjectIdWithTerraDependencies
	ConverterLookupByProjectIdWithTerraDependencies
}

// ConverterByTenantId converts objects based on the relationship to a tenant
type ConverterByTenantId interface {
	ToHclByTenantId(projectId string, dependencies *ResourceDetailsCollection) error
}

// ConvertToHclByResource converts objects directly
type ConvertToHclByResource[C any] interface {
	ToHclByResource(resource C, dependencies *ResourceDetailsCollection) error
}

// ConvertToHclByResource creates a data lookup from the objects
type ConvertToHclLookupByResource[C any] interface {
	ToHclLookupByResource(resource C, dependencies *ResourceDetailsCollection) error
}

// Converter converts all objects in bulk
type Converter interface {
	// ToHcl converts all the resources of a given type to HCL. This is used when converting a space.
	ToHcl(dependencies *ResourceDetailsCollection) error
}
