package terraform

type TerraformTenant struct {
	Type               string                        `hcl:"type,label"`
	Name               string                        `hcl:"name,label"`
	ResourceName       string                        `hcl:"name"`
	Id                 *string                       `hcl:"id"`
	ClonedFromTenantId *string                       `hcl:"cloned_from_tenant_id"`
	Description        *string                       `hcl:"description"`
	TenantTags         []string                      `hcl:"tenant_tags"`
	ProjectEnvironment []TerraformProjectEnvironment `hcl:"project_environment,block"`
}

type TerraformProjectEnvironment struct {
	Environments []string `hcl:"environments"`
	ProjectId    string   `hcl:"project_id"`
}
