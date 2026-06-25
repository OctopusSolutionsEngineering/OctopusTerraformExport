package octopus

type ParentEnvironment struct {
	Id                          string
	SpaceId                     string
	Name                        string
	Slug                        string
	Type                        string
	Description                 string
	SortOrder                   int
	UseGuidedFailure            bool
	EnvironmentTags             []string
	AutomaticDeprovisioningRule *AutomaticDeprovisioningRule
}

type AutomaticDeprovisioningRule struct {
	ExpiryDays  int
	ExpiryHours int
}

func (p ParentEnvironment) GetName() string {
	return p.Name
}

func (p ParentEnvironment) GetId() string {
	return p.Id
}
