package terraform

type TerraformKubernetesEndpointResource struct {
	Type                            string   `hcl:"type,label"`
	Name                            string   `hcl:"name,label"`
	ClusterUrl                      string   `hcl:"cluster_url"`
	Environments                    []string `hcl:"environments"`
	ResourceName                    string   `hcl:"name"`
	Roles                           []string `hcl:"roles"`
	ClusterCertificate              *string  `hcl:"cluster_certificate"`
	DefaultWorkerPoolId             *string  `hcl:"default_worker_pool_id"`
	HealthStatus                    *string  `hcl:"health_status"`
	Id                              *string  `hcl:"id"`
	IsDisabled                      *bool    `hcl:"is_disabled"`
	MachinePolicyId                 *string  `hcl:"machine_policy_id"`
	Namespace                       *string  `hcl:"namespace"`
	OperatingSystem                 *string  `hcl:"operating_system"`
	ProxyId                         *string  `hcl:"proxy_id"`
	RunningInContainer              *string  `hcl:"running_in_container"`
	ShellName                       *string  `hcl:"shell_name"`
	ShellVersion                    *string  `hcl:"shell_version"`
	SkipTlsVerification             *bool    `hcl:"skip_tls_verification"`
	SpaceId                         *string  `hcl:"space_id"`
	Status                          *string  `hcl:"status"`
	StatusSummary                   *string  `hcl:"status_summary"`
	TenantTags                      []string `hcl:"tenant_tags"`
	TenantedDeploymentParticipation string   `hcl:"tenanted_deployment_participation"`
	Tenants                         []string `hcl:"tenants"`
	Thumbprint                      *string  `hcl:"thumbprint"`
	Uri                             *string  `hcl:"uri"`

	Endpoint  TerraformKubernetesEndpoint  `hcl:"endpoint,block"`
	Container TerraformKubernetesContainer `hcl:"container,block"`

	Authentication                      *TerraformAccountAuthentication               `hcl:"authentication,block"`
	AwsAccountAuthentication            *TerraformAwsAccountAuthentication            `hcl:"aws_account_authentication,block"`
	AzureServicePrincipalAuthentication *TerraformAzureServicePrincipalAuthentication `hcl:"azure_service_principal_authentication,block"`
	CertificateAuthentication           *TerraformCertificateAuthentication           `hcl:"certificate_authentication,block"`
	GcpAccountAuthentication            *TerraformGcpAccountAuthentication            `hcl:"gcp_account_authentication,block"`
}

type TerraformAccountAuthentication struct {
	AccountId string `hcl:"account_id"`
}

type TerraformAwsAccountAuthentication struct {
	AccountId                 string  `hcl:"account_id"`
	ClusterName               string  `hcl:"cluster_name"`
	AssumeRole                *bool   `hcl:"assume_role"`
	AssumeRoleExternalId      *string `hcl:"assume_role_external_id"`
	AssumeRoleSessionDuration *string `hcl:"assume_role_session_duration"`
	AssumedRoleArn            *string `hcl:"assumed_role_arn"`
	AssumedRoleSession        *string `hcl:"assumed_role_session"`
	UseInstanceRole           *bool   `hcl:"use_instance_role"`
}

type TerraformAzureServicePrincipalAuthentication struct {
	AccountId            string `hcl:"account_id"`
	ClusterName          string `hcl:"cluster_name"`
	ClusterResourceGroup string `hcl:"cluster_resource_group"`
}

type TerraformCertificateAuthentication struct {
	ClientCertificate *string `hcl:"client_certificate"`
}

type TerraformGcpAccountAuthentication struct {
	AccountId                 string  `hcl:"account_id"`
	ClusterName               string  `hcl:"cluster_name"`
	Project                   string  `hcl:"project"`
	ImpersonateServiceAccount *bool   `hcl:"impersonate_service_account"`
	Region                    *string `hcl:"region"`
	ServiceAccountEmails      *string `hcl:"service_account_emails"`
	UseVmServiceAccount       *bool   `hcl:"use_vm_service_account"`
	Zone                      *string `hcl:"zone"`
}

type TerraformKubernetesContainer struct {
	FeedId string `hcl:"feed_id"`
	Image  string `hcl:"image"`
}

type TerraformKubernetesEndpoint struct {
	CommunicationStyle  string  `hcl:"communication_style"`
	ClusterCertificate  *string `hcl:"cluster_certificate"`
	ClusterUrl          *string `hcl:"cluster_url"`
	Namespace           *string `hcl:"namespace"`
	SkipTlsVerification *bool   `hcl:"skip_tls_verification"`
	DefaultWorkerPoolId *string `hcl:"default_worker_pool_id"`
}
