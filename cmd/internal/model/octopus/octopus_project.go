package octopus

type ProjectConnectivityPolicy struct {
	AllowDeploymentsToNoTargets bool
	ExcludeUnhealthyTargets     bool
	SkipMachineBehavior         string
	TargetRoles                 []string
}

type ProjectCacDeploymentSettings struct {
	Id                              string
	SpaceId                         string
	ProjectId                       string
	ConnectivityPolicy              ProjectConnectivityPolicy
	DefaultGuidedFailureMode        string
	VersioningStrategy              VersioningStrategy
	ReleaseNotesTemplate            *string
	DefaultToSkipIfAlreadyInstalled bool
	DeploymentChangesTemplate       *string
	ForcePackageDownload            bool
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
	NameId
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
	ReleaseNotesTemplate            *string
	VersioningStrategy              VersioningStrategy
	ExtensionSettings               []ExtensionSetting
}

func (p *Project) GetParentId() *string {
	return nil
}

func (p *Project) GetUltimateParent() string {
	return p.Id
}

func (p *Project) HasCacConfigured() bool {
	return p.PersistenceSettings.Credentials.Type == "Reference" ||
		p.PersistenceSettings.Credentials.Type == "Anonymous" ||
		p.PersistenceSettings.Credentials.Type == "UsernamePassword" ||
		p.PersistenceSettings.Credentials.Type == "GitHub"
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

type ExtensionSetting struct {
	ExtensionId string
	Values      map[string]any
}
