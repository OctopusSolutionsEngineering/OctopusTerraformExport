package octopus

type AzureServiceFabricResource struct {
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
	Endpoint                        AzureServiceFabricEndpointResource
}

type AzureServiceFabricEndpointResource struct {
	CommunicationStyle        string
	ConnectionEndpoint        string
	SecurityMode              string
	ServerCertThumbprint      string
	ClientCertVariable        string
	CertificateStoreLocation  string
	CertificateStoreName      string
	AadCredentialType         string
	AadClientCredentialSecret string
	AadUserCredentialUsername string
	AadUserCredentialPassword Secret
	DefaultWorkerPoolId       string
}
