package octopus

// OctopusProcess defines the interface for Octopus processes such as DeploymentProcess and RunbookProcess.
type OctopusProcess interface {
	GetId() string
	GetParentId() string
	GetSteps() []Step
}
