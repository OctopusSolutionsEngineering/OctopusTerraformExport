package octopus

import "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"

type Runbook struct {
	NameId
	Slug                       *string
	Description                *string
	RunbookProcessId           *string
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

func (r *Runbook) GetParentId() *string {
	if r == nil {
		return nil
	}

	return strutil.NilIfEmpty(r.ProjectId)
}

type RunRetentionPolicy struct {
	QuantityToKeep    int
	ShouldKeepForever bool
}
