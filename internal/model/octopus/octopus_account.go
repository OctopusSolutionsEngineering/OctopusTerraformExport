package octopus

type Account struct {
	Id                              string
	Name                            string
	Slug                            *string
	Description                     *string
	SpaceId                         *string
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

	// username
	Username *string

	// google
	JsonKey Secret
}

type Secret struct {
	HasValue bool
	NewValue *string
	Hint     *string
}
