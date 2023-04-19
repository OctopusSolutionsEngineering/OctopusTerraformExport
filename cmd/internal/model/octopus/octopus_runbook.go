package octopus

type Runbook struct {
	Id                         string
	Name                       string
	Slug                       *string
	Description                *string
	RunbookProcessId           string
	PublishedRunbookSnapshotId *string
	ProjectId                  string
	MultiTenancyMode           *string
	ConnectivityPolicy         ProjectConnectivityPolicy
	EnvironmentScope           *string
	Environments               []string
	DefaultGuidedFailureMode   *string
	RunRetentionPolicy         RunRetentionPolicy
	ForcePackageDownload       bool
}

type RunRetentionPolicy struct {
	QuantityToKeep    int
	ShouldKeepForever bool
}
