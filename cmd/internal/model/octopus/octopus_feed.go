package octopus

type Feed struct {
	Id                                string
	SpaceId                           string
	Name                              string
	Slug                              *string
	FeedType                          *string
	FeedUri                           *string
	Username                          *string
	Password                          *Secret
	PackageAcquisitionLocationOptions []string
	RetentionPolicyId                 *string
	DownloadAttempts                  *int
	DownloadRetryBackoffSeconds       *int

	// Docker
	RegistryPath *string
	ApiVersion   *string

	// ECR
	AccessKey *string
	SecretKey *Secret
	Region    *string

	// Nuget
	EnhancedMode bool

	// Artifactory
	Repository  *string
	LayoutRegex *string

	// S3
	UseMachineCredentials *bool
}
