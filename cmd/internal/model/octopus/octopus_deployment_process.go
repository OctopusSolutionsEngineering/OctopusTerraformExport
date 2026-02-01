package octopus

import "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"

type DeploymentProcess struct {
	Id        string
	ProjectId string
	Steps     []Step
}

func (a *DeploymentProcess) GetId() string {
	return a.Id
}

func (a *DeploymentProcess) GetParentId() string {
	return a.ProjectId
}

func (a *DeploymentProcess) GetSteps() []Step {
	return a.Steps
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

func (a *Step) GetId() string {
	return strutil.EmptyIfNil(a.Id)
}

func (a *Step) GetName() string {
	return strutil.EmptyIfNil(a.Name)
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

func (a *Action) GetId() string {
	return a.Id
}

func (a *Action) GetName() string {
	return strutil.EmptyIfNil(a.Name)
}

// GenerateDeploymentProcessId generates a unique identifier for the deployment process. This solves an issue
// where a deployment process has been copied and pasted in Git or cloned via the UI, reusing step and action IDs.
func (a *Action) GenerateDeploymentProcessId(deploymentProcess OctopusProcess) string {
	return deploymentProcess.GetId() + "-" + a.Id
}

// GenerateRunbookProcessId generates a unique identifier for the deployment process. This solves an issue
// where a deployment process has been copied and pasted in Git or cloned via the UI, reusing step and action IDs.
func (a *Action) GenerateRunbookProcessId(runbookProcess OctopusProcess) string {
	return runbookProcess.GetId() + "-" + a.Id
}

type GitDependency struct {
	Name                         *string
	RepositoryUri                *string
	DefaultBranch                *string
	GitCredentialType            *string
	FilePathFilters              []string
	GitCredentialId              *string
	StepPackageInputsReferenceId *string
	GithubConnectionId           *string
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
