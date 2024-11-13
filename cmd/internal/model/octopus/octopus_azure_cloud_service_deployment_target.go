package octopus

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/comparable"
	"strconv"
)

func (r *AzureCloudServiceResource) Compare(other comparable.OctopusResource) comparable.OctopusResourceComparison {
	differences := map[string]comparable.Differences{}

	if otherResource, ok := other.(*AzureCloudServiceResource); ok {
		if r.Name != otherResource.Name {
			differences["Name"] = comparable.Differences{
				SourceValue:      r.Name,
				DestinationValue: otherResource.Name,
			}
		}

		if r.TenantedDeploymentParticipation != otherResource.TenantedDeploymentParticipation {
			differences["TenantedDeploymentParticipation"] = comparable.Differences{
				SourceValue:      r.TenantedDeploymentParticipation,
				DestinationValue: otherResource.TenantedDeploymentParticipation,
			}
		}

		if r.Thumbprint != otherResource.Thumbprint {
			differences["Thumbprint"] = comparable.Differences{
				SourceValue:      r.Thumbprint,
				DestinationValue: otherResource.Thumbprint,
			}
		}

		if r.Uri != otherResource.Uri {
			differences["Uri"] = comparable.Differences{
				SourceValue:      r.Uri,
				DestinationValue: otherResource.Uri,
			}
		}

		if r.IsDisabled != otherResource.IsDisabled {
			differences["IsDisabled"] = comparable.Differences{
				SourceValue:      strconv.FormatBool(r.IsDisabled),
				DestinationValue: strconv.FormatBool(otherResource.IsDisabled),
			}
		}

		if r.MachinePolicyId != otherResource.MachinePolicyId {
			differences["MachinePolicyId"] = comparable.Differences{
				SourceValue:      r.MachinePolicyId,
				DestinationValue: otherResource.MachinePolicyId,
			}
		}

		if r.HealthStatus != otherResource.HealthStatus {
			differences["HealthStatus"] = comparable.Differences{
				SourceValue:      r.HealthStatus,
				DestinationValue: otherResource.HealthStatus,
			}
		}

		if r.HasLatestCalamari != otherResource.HasLatestCalamari {
			differences["HasLatestCalamari"] = comparable.Differences{
				SourceValue:      strconv.FormatBool(r.HasLatestCalamari),
				DestinationValue: strconv.FormatBool(otherResource.HasLatestCalamari),
			}
		}

		if r.StatusSummary != otherResource.StatusSummary {
			differences["StatusSummary"] = comparable.Differences{
				SourceValue:      r.StatusSummary,
				DestinationValue: otherResource.StatusSummary,
			}
		}

		if r.IsInProcess != otherResource.IsInProcess {
			differences["IsInProcess"] = comparable.Differences{
				SourceValue:      strconv.FormatBool(r.IsInProcess),
				DestinationValue: strconv.FormatBool(otherResource.IsInProcess),
			}
		}

		if r.OperatingSystem != otherResource.OperatingSystem {
			differences["OperatingSystem"] = comparable.Differences{
				SourceValue:      r.OperatingSystem,
				DestinationValue: otherResource.OperatingSystem,
			}
		}

		if r.ShellName != otherResource.ShellName {
			differences["ShellName"] = comparable.Differences{
				SourceValue:      r.ShellName,
				DestinationValue: otherResource.ShellName,
			}
		}

		if r.ShellVersion != otherResource.ShellVersion {
			differences["ShellVersion"] = comparable.Differences{
				SourceValue:      r.ShellVersion,
				DestinationValue: otherResource.ShellVersion,
			}
		}

		if r.Architecture != otherResource.Architecture {
			differences["Architecture"] = comparable.Differences{
				SourceValue:      r.Architecture,
				DestinationValue: otherResource.Architecture,
			}
		}

		// Compare Endpoint
		if r.Endpoint.CommunicationStyle != otherResource.Endpoint.CommunicationStyle {
			differences["Endpoint.CommunicationStyle"] = comparable.Differences{
				SourceValue:      r.Endpoint.CommunicationStyle,
				DestinationValue: otherResource.Endpoint.CommunicationStyle,
			}
		}

		if r.Endpoint.DefaultWorkerPoolId != otherResource.Endpoint.DefaultWorkerPoolId {
			differences["Endpoint.DefaultWorkerPoolId"] = comparable.Differences{
				SourceValue:      r.Endpoint.DefaultWorkerPoolId,
				DestinationValue: otherResource.Endpoint.DefaultWorkerPoolId,
			}
		}

		if r.Endpoint.AccountId != otherResource.Endpoint.AccountId {
			differences["Endpoint.AccountId"] = comparable.Differences{
				SourceValue:      r.Endpoint.AccountId,
				DestinationValue: otherResource.Endpoint.AccountId,
			}
		}

		if r.Endpoint.CloudServiceName != otherResource.Endpoint.CloudServiceName {
			differences["Endpoint.CloudServiceName"] = comparable.Differences{
				SourceValue:      r.Endpoint.CloudServiceName,
				DestinationValue: otherResource.Endpoint.CloudServiceName,
			}
		}

		if r.Endpoint.StorageAccountName != otherResource.Endpoint.StorageAccountName {
			differences["Endpoint.StorageAccountName"] = comparable.Differences{
				SourceValue:      r.Endpoint.StorageAccountName,
				DestinationValue: otherResource.Endpoint.StorageAccountName,
			}
		}

		if r.Endpoint.Slot != otherResource.Endpoint.Slot {
			differences["Endpoint.Slot"] = comparable.Differences{
				SourceValue:      r.Endpoint.Slot,
				DestinationValue: otherResource.Endpoint.Slot,
			}
		}

		if r.Endpoint.SwapIfPossible != otherResource.Endpoint.SwapIfPossible {
			differences["Endpoint.SwapIfPossible"] = comparable.Differences{
				SourceValue:      strconv.FormatBool(r.Endpoint.SwapIfPossible),
				DestinationValue: strconv.FormatBool(otherResource.Endpoint.SwapIfPossible),
			}
		}

		if r.Endpoint.UseCurrentInstanceCount != otherResource.Endpoint.UseCurrentInstanceCount {
			differences["Endpoint.UseCurrentInstanceCount"] = comparable.Differences{
				SourceValue:      strconv.FormatBool(r.Endpoint.UseCurrentInstanceCount),
				DestinationValue: strconv.FormatBool(otherResource.Endpoint.UseCurrentInstanceCount),
			}
		}
	}

	return comparable.OctopusResourceComparison{
		SourceResource:                 r,
		DestinationResource:            other,
		Differences:                    differences,
		ChildOctopusResourceComparison: nil,
	}
}

func (r *AzureCloudServiceResource) GetChildResources() []comparable.OctopusResource {
	return []comparable.OctopusResource{}
}

func (r *AzureCloudServiceResource) GetName() string {
	return r.Name
}

type AzureCloudServiceResource struct {
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
	Endpoint                        AzureCloudServiceEndpointResource
}

type AzureCloudServiceEndpointResource struct {
	CommunicationStyle      string
	DefaultWorkerPoolId     string
	AccountId               string
	CloudServiceName        string
	StorageAccountName      string
	Slot                    string
	SwapIfPossible          bool
	UseCurrentInstanceCount bool
}
