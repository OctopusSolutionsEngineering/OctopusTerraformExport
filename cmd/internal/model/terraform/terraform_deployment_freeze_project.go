package terraform

type TerraformDeploymentFreezeProject struct {
	Type               string   `hcl:"type,label"`
	Name               string   `hcl:"name,label"`
	Count              *string  `hcl:"count"`
	Id                 *string  `hcl:"id"`
	DeploymentFreezeId string   `hcl:"deploymentfreeze_id"`
	ProjectId          string   `hcl:"project_id"`
	EnvironmentIds     []string `hcl:"environment_ids"`
}
