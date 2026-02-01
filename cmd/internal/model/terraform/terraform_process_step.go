package terraform

import "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"

type TerraformProcessStep struct {
	Type         string  `hcl:"type,label"`
	Name         string  `hcl:"name,label"`
	Count        *string `hcl:"count"`
	Id           *string `hcl:"id"`
	ResourceName string  `hcl:"name"`
	ResourceType string  `hcl:"type"`
	ProcessId    string  `hcl:"process_id"`
	// ParentId is the ID of the parent step, if this step is a child step.
	ParentId             *string                        `hcl:"parent_id"`
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

func (a *TerraformProcessStep) SetWorkerPoolId(workerPool string) {
	a.WorkerPoolId = strutil.NilIfEmpty(workerPool)
}

type TerraformProcessStepContainer struct {
	FeedId *string `cty:"feed_id"`
	Image  *string `cty:"image"`
}

type TerraformProcessStepGitDependencies struct {
	DefaultBranch      string    `cty:"default_branch"`
	GitCredentialType  string    `cty:"git_credential_type"`
	RepositoryUri      string    `cty:"repository_uri"`
	FilePathFilters    *[]string `cty:"file_path_filters"`
	GitCredentialId    *string   `cty:"git_credential_id"`
	GitHubConnectionId *string   `cty:"github_connection_id"`
}

type TerraformProcessStepPackage struct {
	Id                  *string            `cty:"id"`
	PackageId           string             `cty:"package_id"`
	AcquisitionLocation *string            `cty:"acquisition_location"`
	FeedId              *string            `cty:"feed_id"`
	Properties          *map[string]string `cty:"properties"`
}
