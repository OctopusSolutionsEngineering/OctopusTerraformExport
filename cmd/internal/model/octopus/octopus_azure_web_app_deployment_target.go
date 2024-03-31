package octopus

type AzureWebAppResource struct {
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
	Endpoint                        AzureWebAppEndpointResource
}

type AzureWebAppEndpointResource struct {
	CommunicationStyle  string
	DefaultWorkerPoolId string
	AccountId           string
	ResourceGroupName   string
	WebAppName          string
	WebAppSlotName      string
}
