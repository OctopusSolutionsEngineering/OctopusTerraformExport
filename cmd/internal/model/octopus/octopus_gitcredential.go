package octopus

type GitCredentials struct {
	Id                     string
	SpaceId                string
	Name                   string
	Description            *string
	Details                GitCredentialsDetails
	RepositoryRestrictions RepositoryRestrictions
}

type GitCredentialsDetails struct {
	Type     string
	Username string
}

type RepositoryRestrictions struct {
	Enabled             bool
	AllowedRepositories []string
}
