package octopus

type Channel struct {
	Id                                       string
	SpaceId                                  string
	Name                                     string
	Slug                                     *string
	Description                              *string
	LifecycleId                              *string
	ProjectId                                string
	IsDefault                                bool
	Rules                                    []Rule
	TenantTags                               []string
	EphemeralEnvironmentNameTemplate         *string
	AutomaticEphemeralEnvironmentDeployments *bool
	CustomFieldDefinitions                   []CustomFieldDefinitions
	ParentEnvironmentId                      *string
	Type                                     *string
}

type CustomFieldDefinitions struct {
	FieldName   string
	Description *string
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
