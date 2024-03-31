package octopus

type PollingEndpointResource struct {
	Target

	Id                              string
	Name                            string
	Roles                           []string
	TenantIds                       []string
	TenantTags                      []string
	TenantedDeploymentParticipation string
	Thumbprint                      string
	Uri                             string
	IsDisabled                      bool
	MachinePolicyId                 string
	HealthStatus                    string
	HasLatestCalamari               bool
	StatusSummary                   string
	IsInProcess                     bool
	OperatingSystem                 string
	ShellName                       string
	ShellVersion                    string
	Architecture                    string
	Endpoint                        PollingTentacleEndpointResource
}

// PollingTentacleEndpointResource is based on PollingTentacleEndpointResource from the client library
type PollingTentacleEndpointResource struct {
	CommunicationStyle     string
	Uri                    string
	TentacleVersionDetails TentacleVersionDetails
}
