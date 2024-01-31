package terraform

type TerraformEnvironment struct {
	Type                                   string                                           `hcl:"type,label"`
	Name                                   string                                           `hcl:"name,label"`
	Count                                  *string                                          `hcl:"count"`
	SpaceId                                *string                                          `hcl:"space_id"`
	ResourceName                           string                                           `hcl:"name"`
	Description                            *string                                          `hcl:"description"`
	AllowDynamicInfrastructure             bool                                             `hcl:"allow_dynamic_infrastructure"`
	UseGuidedFailure                       bool                                             `hcl:"use_guided_failure"`
	SortOrder                              int                                              `hcl:"sort_order"`
	JiraExtensionSettings                  *TerraformJiraExtensionSettings                  `hcl:"jira_extension_settings,block"`
	JiraServiceManagementExtensionSettings *TerraformJiraServiceManagementExtensionSettings `hcl:"jira_service_management_extension_settings,block"`
	ServicenowExtensionSettings            *TerraformServicenowExtensionSettings            `hcl:"servicenow_extension_settings,block"`
}

type TerraformJiraExtensionSettings struct {
	EnvironmentType string `hcl:"environment_type"`
}

type TerraformJiraServiceManagementExtensionSettings struct {
	IsEnabled bool `hcl:"is_enabled"`
}

type TerraformServicenowExtensionSettings struct {
	IsEnabled bool `hcl:"is_enabled"`
}
