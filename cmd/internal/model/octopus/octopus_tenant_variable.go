package octopus

type TenantVariable struct {
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
	Variables   map[string]map[string]string
}

type LibraryVariable struct {
	LibraryVariableSetId   string
	LibraryVariableSetName string
	Templates              []Template
	Variables              map[string]string
}
