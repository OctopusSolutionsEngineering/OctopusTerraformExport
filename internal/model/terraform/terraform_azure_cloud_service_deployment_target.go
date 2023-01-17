package terraform

type TerraformAzureCloudServiceDeploymentTarget struct {
	Type               string   `hcl:"type,label"`
	Name               string   `hcl:"name,label"`
	Environments       []string `hcl:"environments"`
	ResourceName       string   `hcl:"name"`
	Roles              []string `hcl:"roles"`
	AccountId          string   `hcl:"account_id"`
	CloudServiceName   string   `hcl:"cloud_service_name"`
	StorageAccountName string   `hcl:"storage_account_name"`

	DefaultWorkerPoolId             *string  `hcl:"default_worker_pool_id"`
	HealthStatus                    *string  `hcl:"health_status"`
	IsDisabled                      *bool    `hcl:"is_disabled"`
	MachinePolicyId                 *string  `hcl:"machine_policy_id"`
	OperatingSystem                 *string  `hcl:"operating_system"`
	ShellName                       *string  `hcl:"shell_name"`
	ShellVersion                    *string  `hcl:"shell_version"`
	Slot                            *string  `hcl:"slot"`
	SpaceId                         *string  `hcl:"space_id"`
	Status                          *string  `hcl:"status"`
	StatusSummary                   *string  `hcl:"status_summary"`
	SwapIfPossible                  *bool    `hcl:"swap_if_possible"`
	TenantTags                      []string `hcl:"tenant_tags"`
	TenantedDeploymentParticipation *string  `hcl:"tenanted_deployment_participation"`
	Tenants                         []string `hcl:"tenants"`
	Thumbprint                      *string  `hcl:"thumbprint"`
	Uri                             *string  `hcl:"uri"`
	UseCurrentInstanceCount         *bool    `hcl:"use_current_instance_count"`
}
