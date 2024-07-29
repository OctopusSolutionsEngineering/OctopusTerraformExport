package octopus

type DeploymentProcess struct {
	Id        string
	ProjectId string
	Steps     []Step
}

type Step struct {
	Id                 *string
	Name               *string
	PackageRequirement *string
	Properties         map[string]string
	Condition          *string
	StartTrigger       *string
	Actions            []Action
}

type Action struct {
	Id                            string
	Name                          *string
	Slug                          *string
	ActionType                    *string
	Notes                         *string
	IsDisabled                    bool
	CanBeUsedForProjectVersioning bool
	IsRequired                    bool
	WorkerPoolId                  string
	Container                     Container
	WorkerPoolVariable            *string
	Environments                  []string
	ExcludedEnvironments          []string
	Channels                      []string
	TenantTags                    []string
	Packages                      []Package
	Condition                     *string
	Properties                    map[string]any
	Inputs                        map[string]any
	GitDependencies               []GitDependency
}

type GitDependency struct {
	Name                         *string
	RepositoryUri                *string
	DefaultBranch                *string
	GitCredentialType            *string
	FilePathFilters              []string
	GitCredentialId              *string
	StepPackageInputsReferenceId *string
}

type Container struct {
	Image  *string
	FeedId *string
}

type Package struct {
	Id                      *string
	Name                    *string
	PackageId               *string
	FeedId                  *string
	AcquisitionLocation     *string
	ExtractDuringDeployment bool
	Properties              map[string]string
}
