package octopus

type Account struct {
	Id                              string
	Name                            string
	Slug                            *string
	Description                     *string
	SpaceId                         string
	EnvironmentIds                  []string
	TenantedDeploymentParticipation *string
	TenantIds                       []string
	TenantTags                      []string
	AccountType                     string

	// token
	Token Secret

	// aws
	AccessKey *string
	SecretKey Secret

	// azure
	SubscriptionNumber                *string
	ClientId                          *string
	TenantId                          *string
	Password                          Secret
	AzureEnvironment                  *string
	ResourceManagementEndpointBaseUri *string
	ActiveDirectoryEndpointBaseUri    *string

	// azure subscription
	ServiceManagementEndpointBaseUri *string
	ServiceManagementEndpointSuffix  *string
	CertificateBytes                 Secret
	CertificateThumbprint            *string

	// username
	Username *string

	// google
	JsonKey Secret

	// aws oidc
	RoleArn                *string
	SessionDuration        *string
	DeploymentSubjectKeys  []string
	HealthCheckSubjectKeys []string
	AccountTestSubjectKeys []string
}

type Secret struct {
	HasValue bool
	NewValue *string
	Hint     *string
}
