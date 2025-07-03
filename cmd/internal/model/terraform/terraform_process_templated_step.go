package terraform

import "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"

// TerraformProcessTemplatesStep defines the resource documented at
// https://registry.terraform.io/providers/OctopusDeploy/octopusdeploy/latest/docs/resources/process_templated_step
type TerraformProcessTemplatedStep struct {
	Type                 string                         `hcl:"type,label"`
	Name                 string                         `hcl:"name,label"`
	Count                *string                        `hcl:"count"`
	ResourceName         string                         `hcl:"name"`
	ProcessId            string                         `hcl:"process_id"`
	TemplateId           string                         `hcl:"template_id"`
	TemplateVersion      string                         `hcl:"template_version"`
	Channels             []string                       `hcl:"channels"`
	Condition            *string                        `hcl:"condition"`
	Container            *TerraformProcessStepContainer `hcl:"container"`
	Environments         []string                       `hcl:"environments"`
	ExcludedEnvironments []string                       `hcl:"excluded_environments"`
	ExecutionProperties  *map[string]string             `hcl:"execution_properties"`
	IsDisabled           *bool                          `hcl:"is_disabled"`
	IsRequired           *bool                          `hcl:"is_required"`
	Notes                *string                        `hcl:"notes"`
	PackageRequirement   *string                        `hcl:"package_requirement"`
	Parameters           *map[string]string             `hcl:"parameters"`
	Properties           *map[string]string             `hcl:"properties"`
	Slug                 *string                        `hcl:"slug"`
	SpaceId              *string                        `hcl:"space_id"`
	StartTrigger         *string                        `hcl:"start_trigger"`
	TenantTags           []string                       `hcl:"tenant_tags"`
	WorkerPoolId         *string                        `hcl:"worker_pool_id"`
	WorkerPoolVariable   *string                        `hcl:"worker_pool_variable"`
}

func (a *TerraformProcessTemplatedStep) SetWorkerPoolId(workerPool string) {
	a.WorkerPoolId = strutil.NilIfEmpty(workerPool)
}
