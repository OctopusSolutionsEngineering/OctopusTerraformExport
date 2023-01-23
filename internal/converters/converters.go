package converters

// ConverterById converts an individual resource by its ID
type ConverterById interface {
	ToHclById(id string, dependencies *ResourceDetailsCollection) error
}

// ConverterByIdWithName converts an individual resource by its ID, and uses the supplied name for the Terraform resource
type ConverterByIdWithName interface {
	ToHclByIdAndName(id string, name string, dependencies *ResourceDetailsCollection) error
}

// ConverterByIdWithNameAndParent converst a resource by its ID, uses the supplied name, and has a reference to its parent
type ConverterByIdWithNameAndParent interface {
	ToHclByIdAndName(id string, name string, parentLookup string, dependencies *ResourceDetailsCollection) error
}

// ConverterByProjectIdWithName converts objects based on their relationship to a project, and uses the supplied name for the Terraform resource
type ConverterByProjectIdWithName interface {
	ToHclByProjectIdAndName(id string, name string, dependencies *ResourceDetailsCollection) error
}

// ConverterByProjectId converts objects based on their relationship to a project
type ConverterByProjectId interface {
	ToHclByProjectId(projectId string, dependencies *ResourceDetailsCollection) error
}

// ConverterByProjectIdWithTerraDependencies converts objects based on their relationship to a project, with manual terraform dependencies
type ConverterByProjectIdWithTerraDependencies interface {
	ToHclByProjectIdWithTerraDependencies(projectId string, terraformDependencies map[string]string, dependencies *ResourceDetailsCollection) error
}

// ConverterByTenantId converts objects based on the relationship to a tenant
type ConverterByTenantId interface {
	ToHclByTenantId(projectId string, dependencies *ResourceDetailsCollection) error
}

// Converter converts all objects in bulk
type Converter interface {
	ToHcl(dependencies *ResourceDetailsCollection) error
}
