package terraform

type TerraformDeploymentFreezeTenant struct {
	Type               string  `hcl:"type,label"`
	Name               string  `hcl:"name,label"`
	Count              *string `hcl:"count"`
	Id                 *string `hcl:"id"`
	DeploymentFreezeId string  `hcl:"deploymentfreeze_id"`
	EnvironmentId      string  `hcl:"environment_id"`
	ProjectId          string  `hcl:"project_id"`
	TenantId           string  `hcl:"tenant_id"`
}
