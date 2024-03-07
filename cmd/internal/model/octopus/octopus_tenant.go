package octopus

type Tenant struct {
	NameId
	SpaceId             *string
	Description         *string
	ClonedFromTenantId  *string
	TenantTags          []string
	ProjectEnvironments map[string][]string
}
