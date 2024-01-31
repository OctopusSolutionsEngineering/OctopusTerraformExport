package terraform

type TerraformEcrFeed struct {
	Type                              string   `hcl:"type,label"`
	Name                              string   `hcl:"name,label"`
	Count                             *string  `hcl:"count"`
	ResourceName                      string   `hcl:"name"`
	AccessKey                         *string  `hcl:"access_key"`
	SecretKey                         *string  `hcl:"secret_key"`
	Region                            *string  `hcl:"region"`
	SpaceId                           *string  `hcl:"space_id"`
	PackageAcquisitionLocationOptions []string `hcl:"package_acquisition_location_options"`
}

type TerraformDockerFeed struct {
	Type                              string   `hcl:"type,label"`
	Name                              string   `hcl:"name,label"`
	Count                             *string  `hcl:"count"`
	ResourceName                      string   `hcl:"name"`
	Password                          *string  `hcl:"password"`
	RegistryPath                      *string  `hcl:"registry_path"`
	Username                          *string  `hcl:"username"`
	ApiVersion                        *string  `hcl:"api_version"`
	SpaceId                           *string  `hcl:"space_id"`
	FeedUri                           *string  `hcl:"feed_uri"`
	PackageAcquisitionLocationOptions []string `hcl:"package_acquisition_location_options"`
}

type TerraformGitHubRepoFeed struct {
	Type                              string   `hcl:"type,label"`
	Name                              string   `hcl:"name,label"`
	Count                             *string  `hcl:"count"`
	ResourceName                      string   `hcl:"name"`
	Password                          *string  `hcl:"password"`
	FeedUri                           *string  `hcl:"feed_uri"`
	DownloadAttempts                  *int     `hcl:"download_attempts"`
	DownloadRetryBackoffSeconds       *int     `hcl:"download_retry_backoff_seconds"`
	Username                          *string  `hcl:"username"`
	SpaceId                           *string  `hcl:"space_id"`
	PackageAcquisitionLocationOptions []string `hcl:"package_acquisition_location_options"`
}

type TerraformHelmFeed struct {
	Type                              string   `hcl:"type,label"`
	Name                              string   `hcl:"name,label"`
	Count                             *string  `hcl:"count"`
	ResourceName                      string   `hcl:"name"`
	Password                          *string  `hcl:"password"`
	FeedUri                           *string  `hcl:"feed_uri"`
	Username                          *string  `hcl:"username"`
	SpaceId                           *string  `hcl:"space_id"`
	PackageAcquisitionLocationOptions []string `hcl:"package_acquisition_location_options"`
}

type TerraformMavenFeed struct {
	Type                              string   `hcl:"type,label"`
	Name                              string   `hcl:"name,label"`
	Count                             *string  `hcl:"count"`
	ResourceName                      string   `hcl:"name"`
	FeedUri                           *string  `hcl:"feed_uri"`
	SpaceId                           *string  `hcl:"space_id"`
	Username                          *string  `hcl:"username"`
	Password                          *string  `hcl:"password"`
	PackageAcquisitionLocationOptions []string `hcl:"package_acquisition_location_options"`
	DownloadAttempts                  *int     `hcl:"download_attempts"`
	DownloadRetryBackoffSeconds       *int     `hcl:"download_retry_backoff_seconds"`
}

type TerraformNuGetFeed struct {
	Type                              string   `hcl:"type,label"`
	Name                              string   `hcl:"name,label"`
	Count                             *string  `hcl:"count"`
	ResourceName                      string   `hcl:"name"`
	FeedUri                           *string  `hcl:"feed_uri"`
	SpaceId                           *string  `hcl:"space_id"`
	Username                          *string  `hcl:"username"`
	Password                          *string  `hcl:"password"`
	IsEnhancedMode                    bool     `hcl:"is_enhanced_mode"`
	PackageAcquisitionLocationOptions []string `hcl:"package_acquisition_location_options"`
	DownloadAttempts                  *int     `hcl:"download_attempts"`
	DownloadRetryBackoffSeconds       *int     `hcl:"download_retry_backoff_seconds"`
}
