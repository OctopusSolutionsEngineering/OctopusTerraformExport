package terraform

type TerraformAzureServiceFabricClusterDeploymentTarget struct {
	Type               string   `hcl:"type,label"`
	Name               string   `hcl:"name,label"`
	Environments       []string `hcl:"environments"`
	ResourceName       string   `hcl:"name"`
	Roles              []string `hcl:"roles"`
	ConnectionEndpoint string   `hcl:"connection_endpoint"`

	AadClientCredentialSecret       *string  `hcl:"aad_client_credential_secret"`
	AadCredentialType               *string  `hcl:"aad_credential_type"`
	AadUserCredentialPassword       *string  `hcl:"aad_user_credential_password"`
	AadUserCredentialUsername       *string  `hcl:"aad_user_credential_username"`
	CertificateStoreLocation        *string  `hcl:"certificate_store_location"`
	CertificateStoreName            *string  `hcl:"certificate_store_name"`
	ClientCertificateVariable       *string  `hcl:"client_certificate_variable"`
	HealthStatus                    *string  `hcl:"health_status"`
	IsDisabled                      *bool    `hcl:"is_disabled"`
	MachinePolicyId                 *string  `hcl:"machine_policy_id"`
	OperatingSystem                 *string  `hcl:"operating_system"`
	SecurityMode                    *string  `hcl:"security_mode"`
	ServerCertificateThumbprint     *string  `hcl:"server_certificate_thumbprint"`
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
