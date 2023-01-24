package octopus

type Channel struct {
	Id          string
	Name        string
	Slug        *string
	Description *string
	LifecycleId string
	ProjectId   string
	IsDefault   bool
	Rules       []Rule
	TenantTags  []string
}

type Rule struct {
	VersionRange   *string
	Tag            *string
	ActionPackages []ActionPackage
	Actions        []string
}

type ActionPackage struct {
	DeploymentAction *string
	PackageReference *string
}
