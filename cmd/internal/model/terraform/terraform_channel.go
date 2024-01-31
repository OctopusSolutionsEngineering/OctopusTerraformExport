package terraform

type TerraformChannel struct {
	Type         string          `hcl:"type,label"`
	Name         string          `hcl:"name,label"`
	Count        *string         `hcl:"count"`
	SpaceId      *string         `hcl:"space_id"`
	LifecycleId  *string         `hcl:"lifecycle_id"`
	ResourceName string          `hcl:"name"`
	Description  *string         `hcl:"description"`
	ProjectId    string          `hcl:"project_id"`
	IsDefault    bool            `hcl:"is_default"`
	Rule         []TerraformRule `hcl:"rule,block"`
	TenantTags   []string        `hcl:"tenant_tags"`
	DependsOn    []string        `hcl:"depends_on"`
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
