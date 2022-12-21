package model

type ProjectConnectivityPolicy struct {
	AllowDeploymentsToNoTargets bool
	ExcludeUnhealthyTargets     bool
	SkipMachineBehavior         *string
}

type Template struct {
	Id              *string
	Name            *string
	Label           *string
	HelpText        *string
	DefaultValue    *string
	DisplaySettings DisplaySettings
}

type DisplaySettings struct {
	OctopusControlType string `json:"Octopus.ControlType"`
}

type Project struct {
	Id                              *string
	Name                            *string
	Description                     *string
	AutoCreateRelease               bool
	DefaultGuidedFailureMode        *string
	DefaultToSkipIfAlreadyInstalled bool
	DiscreteChannelRelease          bool
	IsDisabled                      bool
	IsVersionControlled             bool
	LifecycleId                     *string
	ProjectGroupId                  *string
	DeploymentProcessId             *string
	TenantedDeploymentMode          *string
	ProjectConnectivityPolicy       ProjectConnectivityPolicy
	Templates                       []Template
	VariableSetId                   *string
	// Todo: add service now and jira settings
}
