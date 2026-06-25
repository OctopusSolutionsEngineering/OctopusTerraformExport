package terraform

type TerraformChannel struct {
	Type                             string                  `hcl:"type,label"`
	Name                             string                  `hcl:"name,label"`
	Id                               *string                 `hcl:"id"`
	Count                            *string                 `hcl:"count"`
	SpaceId                          *string                 `hcl:"space_id"`
	LifecycleId                      *string                 `hcl:"lifecycle_id"`
	EphemeralEnvironmentNameTemplate *string                 `hcl:"ephemeral_environment_name_template"`
	ParentEnvironmentId              *string                 `hcl:"parent_environment_id"`
	ResourceName                     string                  `hcl:"name"`
	Description                      *string                 `hcl:"description"`
	ProjectId                        string                  `hcl:"project_id"`
	IsDefault                        bool                    `hcl:"is_default"`
	Rule                             []TerraformRule         `hcl:"rule,block"`
	CustomFieldDefinitions           []CustomFieldDefinition `hcl:"custom_field_definitions,block"`
	TenantTags                       []string                `hcl:"tenant_tags"`
	DependsOn                        []string                `hcl:"depends_on"`
}

type CustomFieldDefinition struct {
	FieldName   string  `hcl:"field_name"`
	Description *string `hcl:"description"`
}

type TerraformRule struct {
	ActionPackage []TerraformActionPackage `hcl:"action_package,block"`
	Tag           *string                  `hcl:"tag"`
	VersionRange  *string                  `hcl:"version_range"`
}

type TerraformActionPackage struct {
	DeploymentAction *string `hcl:"deployment_action"`
	PackageReference *string `hcl:"package_reference"`
}
