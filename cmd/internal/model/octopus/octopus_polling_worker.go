package octopus

type Worker struct {
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
	Endpoint               WorkerEndpoint
}

type WorkerEndpoint struct {
	CommunicationStyle            string
	TentacleVersionDetails        WorkerEndpointTentacleVersionDetails
	Thumbprint                    string
	Uri                           string
	CertificateSignatureAlgorithm string
	Id                            *string
	LastModifiedOn                *string
	LastModifiedBy                *string
	ProxyId                       *string
}

type WorkerEndpointTentacleVersionDetails struct {
	UpgradeLocked    bool
	Version          string
	UpgradeSuggested bool
	UpgradeRequired  bool
	UpgradeAvailable bool
}
