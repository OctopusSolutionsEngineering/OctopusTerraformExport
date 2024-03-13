package terraform

type TerraformListeningTentacleDeploymentTarget struct {
	Type         string   `hcl:"type,label"`
	Name         string   `hcl:"name,label"`
	Id           *string  `hcl:"id"`
	Count        *string  `hcl:"count"`
	Environments []string `hcl:"environments"`
	ResourceName string   `hcl:"name"`
	Roles        []string `hcl:"roles"`
	TentacleUrl  string   `hcl:"tentacle_url"`
	Thumbprint   string   `hcl:"thumbprint"`

	CertificateSignatureAlgorithm   *string                         `hcl:"certificate_signature_algorithm"`
	HealthStatus                    *string                         `hcl:"health_status"`
	IsDisabled                      *bool                           `hcl:"is_disabled"`
	IsInProcess                     *bool                           `hcl:"is_in_process"`
	MachinePolicyId                 *string                         `hcl:"machine_policy_id"`
	OperatingSystem                 *string                         `hcl:"operating_system"`
	ProxyId                         *string                         `hcl:"proxy_id"`
	ShellName                       *string                         `hcl:"shell_name"`
	ShellVersion                    *string                         `hcl:"shell_version"`
	SpaceId                         *string                         `hcl:"space_id"`
	Status                          *string                         `hcl:"status"`
	StatusSummary                   *string                         `hcl:"status_summary"`
	TenantTags                      []string                        `hcl:"tenant_tags"`
	TenantedDeploymentParticipation *string                         `hcl:"tenanted_deployment_participation"`
	Tenants                         []string                        `hcl:"tenants"`
	TentacleVersionDetails          TerraformTentacleVersionDetails `hcl:"tentacle_version_details,block"`
	Uri                             *string                         `hcl:"uri"`
}

type TerraformTentacleVersionDetails struct {
	UpgradeLocked    *bool   `hcl:"upgrade_locked"`
	UpgradeRequired  *bool   `hcl:"upgrade_required"`
	UpgradeSuggested *bool   `hcl:"upgrade_suggested"`
	Version          *string `hcl:"version"`
}
