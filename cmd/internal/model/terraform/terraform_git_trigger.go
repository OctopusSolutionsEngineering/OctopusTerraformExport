package terraform

type TerraformGitTrigger struct {
	Type         string                      `hcl:"type,label"`
	Name         string                      `hcl:"name,label"`
	Id           *string                     `hcl:"id"`
	Count        *string                     `hcl:"count"`
	SpaceId      *string                     `hcl:"space_id"`
	ResourceName string                      `hcl:"name"`
	Description  *string                     `hcl:"description"`
	ProjectId    string                      `hcl:"project_id"`
	ChannelId    string                      `hcl:"channel_id"`
	IsDisabled   *bool                       `hcl:"is_disabled"`
	Sources      []TerraformGitTriggerSource `hcl:"sources"`
}

type TerraformGitTriggerSource struct {
	DeploymentActionSlug string   `cty:"deployment_action_slug"`
	ExcludeFilePaths     []string `cty:"exclude_file_paths"`
	GitDependencyName    string   `cty:"git_dependency_name"`
	IncludeFilePaths     []string `cty:"include_file_paths"`
}
