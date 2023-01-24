package octopus

type Tenant struct {
	Id                  string
	Name                string
	SpaceId             *string
	Description         *string
	ClonedFromTenantId  *string
	TenantTags          []string
	ProjectEnvironments map[string][]string
}
