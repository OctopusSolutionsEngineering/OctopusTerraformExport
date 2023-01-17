package octopus

type CloudRegionResource struct {
	Id                              string
	Name                            string
	EnvironmentIds                  []string
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
	Endpoint                        CloudRegionEndpointResource
}

// CloudRegionEndpointResource is based on PollingTentacleEndpointResource from the client library
type CloudRegionEndpointResource struct {
	CommunicationStyle  string
	DefaultWorkerPoolId string
}
