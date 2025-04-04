package octopus

type SshEndpointResource struct {
	Target

	Id                              string
	Name                            string
	Roles                           []string
	TenantIds                       []string
	TenantTags                      []string
	TenantedDeploymentParticipation string
	Thumbprint                      *string
	Uri                             *string
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
	Endpoint                        SshConnectionDeploymentTargetEndpoint
}

// SshConnectionDeploymentTargetEndpoint is based on SshEndpointResource from the client library
type SshConnectionDeploymentTargetEndpoint struct {
	CommunicationStyle string
	AccountId          string
	Host               string
	Port               int
	Fingerprint        string
	ProxyId            string
	DotNetCorePlatform string
}
