package octopus

type KubernetesEndpointResource struct {
	Id                              string
	Name                            string
	EnvironmentIds                  []string
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
	Endpoint                        KubernetesTentacleEndpointResource
}

// KubernetesTentacleEndpointResource is based on KubernetesEndpointResource from the client library
type KubernetesTentacleEndpointResource struct {
	CommunicationStyle     string
	ClusterCertificate     *string
	ClusterCertificatePath *string
	ClusterUrl             *string
	Namespace              *string
	SkipTlsVerification    *string
	ProxyId                *string
	DefaultWorkerPoolId    *string
	Container              DeploymentActionContainerResource
	Authentication         AuthenticationResource
}

type DeploymentActionContainerResource struct {
	Image  string
	FeedId string
}

type AuthenticationResource struct {
	AuthenticationType string

	// KubernetesCertificateAuthenticationResource
	ClientCertificate *string

	// KubernetesPodServiceAuthenticationResource
	TokenPath *string

	// KubernetesStandardAccountAuthenticationResource
	AccountId *string

	// KubernetesAzureAuthenticationResource and KubernetesAwsAuthenticationResource
	ClusterName          *string
	ClusterResourceGroup *string

	// KubernetesAwsAuthenticationResource
	UseInstanceRole                  *bool
	AssumeRole                       *bool
	AssumedRoleArn                   *string
	AssumedRoleSession               *string
	AssumeRoleSessionDurationSeconds *string
	AssumeRoleExternalId             *string

	// KubernetesGoogleCloudAuthenticationResource
	UseVmServiceAccount       *bool
	ImpersonateServiceAccount *bool
	ServiceAccountEmails      *string
	Project                   *string
	Region                    *string
	Zone                      *string
}
