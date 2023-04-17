package terraform

type TerraformAzureCloudServiceDeploymentTargetsData struct {
	Type           string   `hcl:"type,label"`
	Name           string   `hcl:"name,label"`
	ResourceName   *string  `hcl:"name"`
	Ids            []string `hcl:"ids"`
	PartialName    *string  `hcl:"partial_name"`
	Skip           int      `hcl:"skip"`
	Take           int      `hcl:"take"`
	HealthStatuses []string `hcl:"health_statuses"`
}
