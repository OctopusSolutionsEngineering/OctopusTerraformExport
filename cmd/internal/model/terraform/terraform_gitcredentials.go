package terraform

type TerraformGitCredentials struct {
	Type                   string                                       `hcl:"type,label"`
	Name                   string                                       `hcl:"name,label"`
	Count                  *string                                      `hcl:"count"`
	Id                     *string                                      `hcl:"id"`
	SpaceId                *string                                      `hcl:"space_id"`
	Description            *string                                      `hcl:"description"`
	ResourceName           string                                       `hcl:"name"`
	ResourceType           string                                       `hcl:"type"`
	Username               string                                       `hcl:"username"`
	Password               string                                       `hcl:"password"`
	RepositoryRestrictions TerraformGitCredentialsRepositoryRestriction `hcl:"repository_restrictions"`
}

type TerraformGitCredentialsRepositoryRestriction struct {
	AllowedRepositories []string `cty:"allowed_repositories"`
	Enabled             bool     `cty:"enabled"`
}
