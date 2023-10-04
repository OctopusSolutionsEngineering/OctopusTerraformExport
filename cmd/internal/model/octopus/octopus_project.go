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
	DefaultValue    any
	DisplaySettings map[string]string
}

func (t Template) GetDefaultValueString() *string {
	// Default value is either a string or an object defining the secret variable.
	// We are only interested in defining the string value, as we can not get the secret value through the API.
	defaultValue, ok := t.DefaultValue.(string)
	if !ok {
		return nil
	}
	return &defaultValue
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

func (t Project) HasCacConfigured() bool {
	return t.PersistenceSettings.Credentials.Type == "Reference" ||
		t.PersistenceSettings.Credentials.Type == "Anonymous" ||
		t.PersistenceSettings.Credentials.Type == "UsernamePassword"
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
