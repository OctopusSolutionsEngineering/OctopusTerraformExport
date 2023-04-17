package terraform

type TerraformDeploymentTargetsData struct {
	Type                string   `hcl:"type,label"`
	Name                string   `hcl:"name,label"`
	ResourceName        *string  `hcl:"name"`
	Ids                 []string `hcl:"ids"`
	PartialName         *string  `hcl:"partial_name"`
	Skip                int      `hcl:"skip"`
	Take                int      `hcl:"take"`
	HealthStatuses      []string `hcl:"health_statuses"`
	CommunicationStyles []string `hcl:"communication_styles"`
	DeploymentId        *string  `hcl:"deployment_id"`
	Environments        []string `hcl:"environments"`
	Roles               []string `hcl:"roles"`
	ShellNames          []string `hcl:"shell_names"`
	TenantTags          []string `hcl:"tenant_tags"`
	Tenants             []string `hcl:"tenants"`
	Thumbprint          *string  `hcl:"thumbprint"`
	IsDisabled          *bool    `hcl:"is_disabled"`
}
