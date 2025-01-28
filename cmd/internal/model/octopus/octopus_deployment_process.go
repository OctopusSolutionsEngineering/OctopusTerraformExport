package octopus

import "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"

type DeploymentProcess struct {
	Id        string
	ProjectId string
	Steps     []Step
}

type Step struct {
	Id                 *string
	Name               *string
	PackageRequirement *string
	Properties         map[string]string
	Condition          *string
	StartTrigger       *string
	Actions            []Action
}

// GenerateDeploymentProcessId generates a unique identifier for the deployment process. This solves an issue
// where a deployment process has been copied and pasted in Git or cloned via the UI, reusing step and action IDs.
func (a *Step) GenerateDeploymentProcessId(deploymentProcess *DeploymentProcess) string {
	return deploymentProcess.Id + "-" + strutil.EmptyIfNil(a.Id)
}

// GenerateRunbookProcessId generates a unique identifier for the deployment process. This solves an issue
// where a deployment process has been copied and pasted in Git or cloned via the UI, reusing step and action IDs.
func (a *Step) GenerateRunbookProcessId(runbookProcess *RunbookProcess) string {
	return runbookProcess.Id + "-" + strutil.EmptyIfNil(a.Id)
}

type Action struct {
	Id                            string
	Name                          *string
	Slug                          *string
	ActionType                    *string
	Notes                         *string
	IsDisabled                    bool
	CanBeUsedForProjectVersioning bool
	IsRequired                    bool
	WorkerPoolId                  string
	Container                     Container
	WorkerPoolVariable            *string
	Environments                  []string
	ExcludedEnvironments          []string
	Channels                      []string
	TenantTags                    []string
	Packages                      []Package
	Condition                     *string
	Properties                    map[string]any
	Inputs                        map[string]any
	GitDependencies               []GitDependency
}

// GenerateDeploymentProcessId generates a unique identifier for the deployment process. This solves an issue
// where a deployment process has been copied and pasted in Git or cloned via the UI, reusing step and action IDs.
func (a *Action) GenerateDeploymentProcessId(deploymentProcess *DeploymentProcess) string {
	return deploymentProcess.Id + "-" + a.Id
}

// GenerateRunbookProcessId generates a unique identifier for the deployment process. This solves an issue
// where a deployment process has been copied and pasted in Git or cloned via the UI, reusing step and action IDs.
func (a *Action) GenerateRunbookProcessId(runbookProcess *RunbookProcess) string {
	return runbookProcess.Id + "-" + a.Id
}

type GitDependency struct {
	Name                         *string
	RepositoryUri                *string
	DefaultBranch                *string
	GitCredentialType            *string
	FilePathFilters              []string
	GitCredentialId              *string
	StepPackageInputsReferenceId *string
}

type Container struct {
	Image  *string
	FeedId *string
}

type Package struct {
	Id                      *string
	Name                    *string
	PackageId               *string
	FeedId                  *string
	AcquisitionLocation     *string
	ExtractDuringDeployment bool
	Properties              map[string]string
}
