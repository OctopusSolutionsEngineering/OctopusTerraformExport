package terraform

type TerraformTenant struct {
	Type               string   `hcl:"type,label"`
	Name               string   `hcl:"name,label"`
	Count              *string  `hcl:"count"`
	ResourceName       string   `hcl:"name"`
	Id                 *string  `hcl:"id"`
	SpaceId            *string  `hcl:"space_id"`
	ClonedFromTenantId *string  `hcl:"cloned_from_tenant_id"`
	Description        *string  `hcl:"description"`
	TenantTags         []string `hcl:"tenant_tags"`
	DependsOn          []string `hcl:"depends_on"`
}

type TerraformProjectEnvironment struct {
	Environments []string `hcl:"environments"`
	ProjectId    string   `hcl:"project_id"`
}
