package terraform

type TerraformProcessStep struct {
	Type                 string                         `hcl:"type,label"`
	Name                 string                         `hcl:"name,label"`
	Count                *string                        `hcl:"count"`
	Id                   *string                        `hcl:"id"`
	ResourceName         string                         `hcl:"name"`
	ResourceType         string                         `hcl:"type"`
	ProcessId            string                         `hcl:"process_id"`
	Channels             []string                       `hcl:"channels"`
	Condition            *string                        `hcl:"condition"`
	Container            *TerraformProcessStepContainer `hcl:"container"`
	Environments         []string                       `hcl:"environments"`
	ExcludedEnvironments []string                       `hcl:"excluded_environments"`
	// ExecutionProperties are properties associated with the step.
	ExecutionProperties *map[string]string                              `hcl:"execution_properties"`
	GitDependencies     *map[string]TerraformProcessStepGitDependencies `hcl:"git_dependencies"`
	IsDisabled          *bool                                           `hcl:"is_disabled"`
	IsRequired          *bool                                           `hcl:"is_required"`
	Notes               *string                                         `hcl:"notes"`
	PackageRequirement  *string                                         `hcl:"package_requirement"`
	Packages            *map[string]TerraformProcessStepPackage         `hcl:"packages"`
	PrimaryPackage      *TerraformProcessStepPackage                    `hcl:"primary_package"`
	// Properties are properties associated with the first action.
	Properties         *map[string]string `hcl:"properties"`
	Slug               *string            `hcl:"slug"`
	SpaceId            *string            `hcl:"space_id"`
	StartTrigger       *string            `hcl:"start_trigger"`
	TenantTags         []string           `hcl:"tenant_tags"`
	WorkerPoolId       *string            `hcl:"worker_pool_id"`
	WorkerPoolVariable *string            `hcl:"worker_pool_variable"`
}

type TerraformProcessStepContainer struct {
	Type   string  `hcl:"type,label"`
	Name   string  `hcl:"name,label"`
	FeedId *string `hcl:"feed_id"`
	Image  *string `hcl:"image"`
}

type TerraformProcessStepGitDependencies struct {
	DefaultBranch     string    `hcl:"default_branch"`
	GitCredentialType string    `hcl:"git_credential_type"`
	RepositoryUri     string    `hcl:"repository_uri"`
	FilePathFilters   *[]string `hcl:"file_path_filters"`
	GitCredentialId   *string   `hcl:"git_credential_id"`
}

type TerraformProcessStepPackage struct {
	Id                  *string            `hcl:"id"`
	PackageId           string             `hcl:"package_id"`
	AcquisitionLocation *string            `hcl:"acquisition_location"`
	FeedId              *string            `hcl:"feed_id"`
	Properties          *map[string]string `hcl:"properties"`
}
