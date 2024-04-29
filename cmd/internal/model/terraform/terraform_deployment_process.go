package terraform

type TerraformDeploymentProcess struct {
	Type      string                          `hcl:"type,label"`
	Name      string                          `hcl:"name,label"`
	Id        *string                         `hcl:"id"`
	Count     *string                         `hcl:"count"`
	ProjectId string                          `hcl:"project_id"`
	Step      []TerraformStep                 `hcl:"step,block"`
	Lifecycle *TerraformLifecycleMetaArgument `hcl:"lifecycle,block"`
}

type TerraformStep struct {
	Condition           *string           `hcl:"condition"`
	ConditionExpression *string           `hcl:"condition_expression"`
	Name                *string           `hcl:"name"`
	PackageRequirement  *string           `hcl:"package_requirement"`
	StartTrigger        *string           `hcl:"start_trigger"`
	Action              []TerraformAction `hcl:"action,block"`
	Properties          map[string]string `hcl:"properties"`
	TargetRoles         []string          `hcl:"target_roles"`
}

type TerraformAction struct {
	ActionType                    *string                  `hcl:"action_type"`
	Name                          *string                  `hcl:"name"`
	Notes                         *string                  `hcl:"notes"`
	Condition                     *string                  `hcl:"condition"`
	RunOnServer                   bool                     `hcl:"run_on_server"`
	IsDisabled                    bool                     `hcl:"is_disabled"`
	CanBeUsedForProjectVersioning bool                     `hcl:"can_be_used_for_project_versioning"`
	IsRequired                    bool                     `hcl:"is_required"`
	WorkerPoolId                  *string                  `hcl:"worker_pool_id"`
	WorkerPoolVariable            *string                  `hcl:"worker_pool_variable"`
	Properties                    map[string]string        `hcl:"properties"`
	Container                     *TerraformContainer      `hcl:"container,block"`
	Environments                  []string                 `hcl:"environments"`
	ExcludedEnvironments          []string                 `hcl:"excluded_environments"`
	Channels                      []string                 `hcl:"channels"`
	TenantTags                    []string                 `hcl:"tenant_tags"`
	Package                       []TerraformPackage       `hcl:"package,block"`
	PrimaryPackage                *TerraformPackage        `hcl:"primary_package,block"`
	Features                      []string                 `hcl:"features"`
	GitDependencies               []TerraformGitDependency `hcl:"git_dependency,block"`
}

type TerraformGitDependency struct {
	RepositoryUri     *string `hcl:"repository_uri"`
	DefaultBranch     *string `hcl:"default_branch"`
	GitCredentialType *string `hcl:"git_credential_type"`
	GitCredentialID   *string `hcl:"git_credential_id"`
}

type TerraformContainer struct {
	FeedId *string `hcl:"feed_id"`
	Image  *string `hcl:"image"`
}

type TerraformPackage struct {
	Name                    *string           `hcl:"name"`
	PackageID               *string           `hcl:"package_id"`
	AcquisitionLocation     *string           `hcl:"acquisition_location"`
	ExtractDuringDeployment *bool             `hcl:"extract_during_deployment"`
	FeedId                  *string           `hcl:"feed_id"`
	Id                      *string           `hcl:"id"`
	Properties              map[string]string `hcl:"properties"`
}
