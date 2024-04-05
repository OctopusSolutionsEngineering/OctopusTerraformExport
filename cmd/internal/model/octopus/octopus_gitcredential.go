package octopus

type GitCredentials struct {
	Id          string
	SpaceId     string
	Name        string
	Description *string
	Details     GitCredentialsDetails
}

type GitCredentialsDetails struct {
	Type     string
	Username string
}
