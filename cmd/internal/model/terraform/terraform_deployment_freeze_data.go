package terraform

type TerraformDeploymentFreezeData struct {
	Type            string   `hcl:"type,label"`
	Name            string   `hcl:"name,label"`
	Ids             []string `hcl:"ids"`
	EnvironmentIds  []string `hcl:"environment_ids"`
	ProjectIds      []string `hcl:"project_ids"`
	Status          *string  `hcl:"status"`
	TenantIds       []string `hcl:"tenant_ids"`
	IncludeComplete *bool    `hcl:"include_complete"`
	PartialName     string   `hcl:"partial_name"`
	Skip            int      `hcl:"skip"`
	Take            int      `hcl:"take"`
}
