package octopus

type ProjectConnectivityPolicy struct {
	AllowDeploymentsToNoTargets bool
	ExcludeUnhealthyTargets     bool
	SkipMachineBehavior         string
}

type Template struct {
	Id              string
	Name            *string
	Label           *string
	HelpText        *string
	DefaultValue    *string
	DisplaySettings map[string]string
}

type Project struct {
	Id                              string
	Name                            string
	Slug                            *string
	Description                     *string
	AutoCreateRelease               bool
	DefaultGuidedFailureMode        *string
	DefaultToSkipIfAlreadyInstalled bool
	DiscreteChannelRelease          bool
	IsDisabled                      bool
	IsVersionControlled             bool
	LifecycleId                     string
	ProjectGroupId                  string
	DeploymentProcessId             *string
	TenantedDeploymentMode          *string
	ProjectConnectivityPolicy       ProjectConnectivityPolicy
	Templates                       []Template
	VariableSetId                   *string
	IncludedLibraryVariableSetIds   []string
	PersistenceSettings             PersistenceSettings
	VersioningStrategy              VersioningStrategy
	// Todo: add service now and jira settings
}

type PersistenceSettings struct {
	Type                        string
	Url                         string
	Credentials                 Credentials
	DefaultBranch               string
	BasePath                    string
	ProtectedDefaultBranch      bool
	ProtectedBranchNamePatterns []string
	ConversionState             ConversionState
}

type Credentials struct {
	Type     string
	Id       string
	Username string
}

type ConversionState struct {
	VariablesAreInGit bool
}

type VersioningStrategy struct {
	Template           string
	DonorPackageStepId *string
	DonorPackage       *DonorPackage
}

type DonorPackage struct {
	DeploymentAction *string
	PackageReference *string
}
