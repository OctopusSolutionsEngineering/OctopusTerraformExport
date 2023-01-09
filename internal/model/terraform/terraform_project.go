package terraform

type TerraformProject struct {
	Type                            string              `hcl:"type,label"`
	Name                            string              `hcl:"name,label"`
	ResourceName                    string              `hcl:"name"`
	AutoCreateRelease               bool                `hcl:"auto_create_release"`
	DefaultGuidedFailureMode        *string             `hcl:"default_guided_failure_mode"`
	DefaultToSkipIfAlreadyInstalled bool                `hcl:"default_to_skip_if_already_installed"`
	Description                     *string             `hcl:"description"`
	DiscreteChannelRelease          bool                `hcl:"discrete_channel_release"`
	IsDisabled                      bool                `hcl:"is_disabled"`
	IsVersionControlled             bool                `hcl:"is_version_controlled"`
	LifecycleId                     string              `hcl:"lifecycle_id"`
	ProjectGroupId                  string              `hcl:"project_group_id"`
	IncludedLibraryVariableSets     []string            `hcl:"included_library_variable_sets"`
	TenantedDeploymentParticipation *string             `hcl:"tenanted_deployment_participation"`
	Template                        []TerraformTemplate `hcl:"template,block"`
}

type TerraformTemplate struct {
	Name            *string           `cty:"name"`
	Label           *string           `cty:"label"`
	HelpText        *string           `cty:"help_text"`
	DefaultValue    *string           `cty:"default_value"`
	DisplaySettings map[string]string `cty:"display_settings"`
}
