package terraform

type TerraformOfflineDropDeploymentTarget struct {
	Type                  string   `hcl:"type,label"`
	Name                  string   `hcl:"name,label"`
	ApplicationsDirectory string   `hcl:"applications_directory"`
	WorkingDirectory      string   `hcl:"working_directory"`
	Environments          []string `hcl:"environments"`
	ResourceName          string   `hcl:"name"`
	Roles                 []string `hcl:"roles"`

	HealthStatus                    *string  `hcl:"health_status"`
	IsDisabled                      *bool    `hcl:"is_disabled"`
	MachinePolicyId                 *string  `hcl:"machine_policy_id"`
	OperatingSystem                 *string  `hcl:"operating_system"`
	ShellName                       *string  `hcl:"shell_name"`
	ShellVersion                    *string  `hcl:"shell_version"`
	SpaceId                         *string  `hcl:"space_id"`
	Status                          *string  `hcl:"status"`
	StatusSummary                   *string  `hcl:"status_summary"`
	TenantTags                      []string `hcl:"tenant_tags"`
	TenantedDeploymentParticipation *string  `hcl:"tenanted_deployment_participation"`
	Tenants                         []string `hcl:"tenants"`
	Thumbprint                      *string  `hcl:"thumbprint"`
	Uri                             *string  `hcl:"uri"`
}
