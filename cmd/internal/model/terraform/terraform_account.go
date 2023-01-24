package terraform

type TerraformAwsAccount struct {
	Type                            string   `hcl:"type,label"`
	Name                            string   `hcl:"name,label"`
	SpaceId                         *string  `hcl:"space_id"`
	ResourceName                    string   `hcl:"name"`
	Description                     *string  `hcl:"description"`
	Environments                    []string `hcl:"environments"`
	TenantTags                      []string `hcl:"tenant_tags"`
	Tenants                         []string `hcl:"tenants"`
	TenantedDeploymentParticipation *string  `hcl:"tenanted_deployment_participation"`
	AccessKey                       *string  `hcl:"access_key"`
	SecretKey                       *string  `hcl:"secret_key"`
}

type TerraformAzureServicePrincipal struct {
	Type                            string   `hcl:"type,label"`
	Name                            string   `hcl:"name,label"`
	SpaceId                         *string  `hcl:"space_id"`
	ResourceName                    string   `hcl:"name"`
	Description                     *string  `hcl:"description"`
	Environments                    []string `hcl:"environments"`
	TenantTags                      []string `hcl:"tenant_tags"`
	Tenants                         []string `hcl:"tenants"`
	TenantedDeploymentParticipation *string  `hcl:"tenanted_deployment_participation"`
	ApplicationId                   *string  `hcl:"application_id"`
	Password                        *string  `hcl:"password"`
	SubscriptionId                  *string  `hcl:"subscription_id"`
	TenantId                        *string  `hcl:"tenant_id"`
	AzureEnvironment                *string  `hcl:"azure_environment"`
	ResourceManagerEndpoint         *string  `hcl:"resource_manager_endpoint"`
}

type TerraformAzureSubscription struct {
	Type                            string   `hcl:"type,label"`
	Name                            string   `hcl:"name,label"`
	SpaceId                         *string  `hcl:"space_id"`
	ResourceName                    string   `hcl:"name"`
	Description                     *string  `hcl:"description"`
	Environments                    []string `hcl:"environments"`
	TenantTags                      []string `hcl:"tenant_tags"`
	Tenants                         []string `hcl:"tenants"`
	TenantedDeploymentParticipation *string  `hcl:"tenanted_deployment_participation"`
	// ManagementEndpoint is required, even if it is a blank string
	ManagementEndpoint string `hcl:"management_endpoint"`
	// StorageEndpointSuffix is required, even if it is a blank string
	StorageEndpointSuffix string  `hcl:"storage_endpoint_suffix"`
	SubscriptionId        *string `hcl:"subscription_id"`
	AzureEnvironment      *string `hcl:"azure_environment"`
	Certificate           *string `hcl:"certificate"`
	CertificateThumbprint *string `hcl:"certificate_thumbprint"`
}

type TerraformGcpAccount struct {
	Type                            string   `hcl:"type,label"`
	Name                            string   `hcl:"name,label"`
	SpaceId                         *string  `hcl:"space_id"`
	ResourceName                    string   `hcl:"name"`
	Description                     *string  `hcl:"description"`
	Environments                    []string `hcl:"environments"`
	TenantTags                      []string `hcl:"tenant_tags"`
	Tenants                         []string `hcl:"tenants"`
	TenantedDeploymentParticipation *string  `hcl:"tenanted_deployment_participation"`
	JsonKey                         *string  `hcl:"json_key"`
}

type TerraformSshAccount struct {
	Type                            string   `hcl:"type,label"`
	Name                            string   `hcl:"name,label"`
	SpaceId                         *string  `hcl:"space_id"`
	ResourceName                    string   `hcl:"name"`
	Description                     *string  `hcl:"description"`
	Environments                    []string `hcl:"environments"`
	TenantTags                      []string `hcl:"tenant_tags"`
	Tenants                         []string `hcl:"tenants"`
	TenantedDeploymentParticipation *string  `hcl:"tenanted_deployment_participation"`
	PrivateKeyFile                  *string  `hcl:"private_key_file"`
	Username                        *string  `hcl:"username"`
	PrivateKeyPassphrase            *string  `hcl:"private_key_passphrase"`
}

type TerraformTokenAccount struct {
	Type                            string   `hcl:"type,label"`
	Name                            string   `hcl:"name,label"`
	SpaceId                         *string  `hcl:"space_id"`
	ResourceName                    string   `hcl:"name"`
	Description                     *string  `hcl:"description"`
	Environments                    []string `hcl:"environments"`
	TenantTags                      []string `hcl:"tenant_tags"`
	Tenants                         []string `hcl:"tenants"`
	TenantedDeploymentParticipation *string  `hcl:"tenanted_deployment_participation"`
	Token                           *string  `hcl:"token"`
}

type TerraformUsernamePasswordAccount struct {
	Type                            string   `hcl:"type,label"`
	Name                            string   `hcl:"name,label"`
	SpaceId                         *string  `hcl:"space_id"`
	ResourceName                    string   `hcl:"name"`
	Description                     *string  `hcl:"description"`
	Environments                    []string `hcl:"environments"`
	TenantTags                      []string `hcl:"tenant_tags"`
	Tenants                         []string `hcl:"tenants"`
	TenantedDeploymentParticipation *string  `hcl:"tenanted_deployment_participation"`
	Username                        *string  `hcl:"username"`
	Password                        *string  `hcl:"password"`
}
