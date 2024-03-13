package terraform

type TerraformAzureWebAppDeploymentTarget struct {
	Type              string   `hcl:"type,label"`
	Name              string   `hcl:"name,label"`
	Id                *string  `hcl:"id"`
	Count             *string  `hcl:"count"`
	Environments      []string `hcl:"environments"`
	ResourceName      string   `hcl:"name"`
	Roles             []string `hcl:"roles"`
	AccountId         string   `hcl:"account_id"`
	ResourceGroupName string   `hcl:"resource_group_name"`
	WebAppName        string   `hcl:"web_app_name"`

	HealthStatus                    *string                                       `hcl:"health_status"`
	IsDisabled                      *bool                                         `hcl:"is_disabled"`
	MachinePolicyId                 *string                                       `hcl:"machine_policy_id"`
	OperatingSystem                 *string                                       `hcl:"operating_system"`
	ShellName                       *string                                       `hcl:"shell_name"`
	ShellVersion                    *string                                       `hcl:"shell_version"`
	SpaceId                         *string                                       `hcl:"space_id"`
	Status                          *string                                       `hcl:"status"`
	StatusSummary                   *string                                       `hcl:"status_summary"`
	TenantTags                      []string                                      `hcl:"tenant_tags"`
	TenantedDeploymentParticipation *string                                       `hcl:"tenanted_deployment_participation"`
	Tenants                         []string                                      `hcl:"tenants"`
	Thumbprint                      *string                                       `hcl:"thumbprint"`
	Uri                             *string                                       `hcl:"uri"`
	WebAppSlotName                  *string                                       `hcl:"web_app_slot_name"`
	Endpoint                        *TerraformAzureWebAppDeploymentTargetEndpoint `hcl:"endpoint,block"`
}

type TerraformAzureWebAppDeploymentTargetEndpoint struct {
	DefaultWorkerPoolId *string `hcl:"default_worker_pool_id"`
	CommunicationStyle  string  `hcl:"communication_style"`
}
