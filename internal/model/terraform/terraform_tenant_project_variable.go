package terraform

type TerraformTenantProjectVariable struct {
	Type          string  `hcl:"type,label"`
	Name          string  `hcl:"name,label"`
	Id            *string `hcl:"id"`
	EnvironmentId string  `hcl:"environment_id"`
	ProjectId     string  `hcl:"project_id"`
	TemplateId    string  `hcl:"template_id"`
	TenantId      string  `hcl:"tenant_id"`
	Value         *string `hcl:"value"`
}
