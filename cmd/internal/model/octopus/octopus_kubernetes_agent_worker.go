package octopus

type KubernetesAgentWorker struct {
	Id                     string
	SpaceId                string
	WorkerPoolIds          []string
	Name                   string
	Thumbprint             string
	Uri                    *string
	IsDisabled             bool
	MachinePolicyId        string
	HealthStatus           string
	HasLatestCalamari      bool
	StatusSummary          string
	IsInProcess            bool
	OperatingSystem        string
	ShellName              string
	ShellVersion           string
	Architecture           string
	Slug                   string
	SkipInitialHealthCheck bool
	Endpoint               KubernetesAgentWorkerEndpoint
}

type KubernetesAgentWorkerEndpoint struct {
	CommunicationStyle            string
	TentacleEndpointConfiguration KubernetesAgentWorkerEndpointTentacleEndpointConfiguration
	KubernetesAgentDetails        KubernetesAgentWorkerEndpointKubernetesAgentDetails
	UpgradeLocked                 bool
	DefaultNamespace              *string
	od                            *string
	LastModifiedOn                *string
	LastModifiedBy                *string
}

type KubernetesAgentWorkerEndpointTentacleEndpointConfiguration struct {
	CommunicationMode             string
	Thumbprint                    string
	Uri                           string
	CertificateSignatureAlgorithm *string
}

type KubernetesAgentWorkerEndpointKubernetesAgentDetails struct {
	AgentVersion        string
	TentacleVersion     string
	UpgradeStatus       string
	HelmReleaseName     string
	KubernetesNamespace string
}
