package octopus

type KubernetesAgentResource struct {
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
	Endpoint                        KubernetesAgentEndpointResource
}

type KubernetesAgentEndpointResource struct {
	CommunicationStyle            string
	TentacleEndpointConfiguration KubernetesAgentTentacleEndpointConfiguration
	KubernetesAgentDetails        KubernetesAgentDetails
	UpgradeLocked                 bool
	DefaultNamespace              *string
	Id                            *string
	LastModifiedOn                *string
	LastModifiedBy                *string
}

type KubernetesAgentTentacleEndpointConfiguration struct {
	CommunicationMode             string
	Thumbprint                    string
	Uri                           string
	CertificateSignatureAlgorithm *string
}

type KubernetesAgentDetails struct {
	AgentVersion        string
	TentacleVersion     string
	UpgradeStatus       string
	HelmReleaseName     string
	KubernetesNamespace string
}
