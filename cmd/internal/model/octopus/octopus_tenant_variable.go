package octopus

type TenantVariable struct {
	Id               string
	TenantId         string
	SpaceId          string
	TenantName       string
	ConcurrencyToken string
	ProjectVariables map[string]ProjectVariable
	LibraryVariables map[string]LibraryVariable
}

type ProjectVariable struct {
	ProjectId   string
	ProjectName string
	Templates   []Template
	Variables   map[string]map[string]any // The value of a secret is either a string or an object indicating the state of the secret
}

type LibraryVariable struct {
	LibraryVariableSetId   string
	LibraryVariableSetName string
	Templates              []Template
	Variables              map[string]any // The value of a secret is either a string or an object indicating the state of the secret
}
