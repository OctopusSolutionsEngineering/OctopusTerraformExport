package terraform

type TerraformTenantProjectEnvironment struct {
	Type           string   `hcl:"type,label"`
	Name           string   `hcl:"name,label"`
	Count          *string  `hcl:"count"`
	SpaceId        *string  `hcl:"space_id"`
	TenantId       string   `hcl:"tenant_id"`
	ProjectId      string   `hcl:"project_id"`
	EnvironmentIds []string `hcl:"environment_ids"`
}
