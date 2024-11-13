package octopus

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/comparable"
	"strconv"
)

func (r *AzureServiceFabricResource) Compare(other comparable.OctopusResource) comparable.OctopusResourceComparison {
	differences := map[string]comparable.Differences{}

	if otherResource, ok := other.(*AzureServiceFabricResource); ok {
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

		if r.Endpoint.ConnectionEndpoint != otherResource.Endpoint.ConnectionEndpoint {
			differences["Endpoint.ConnectionEndpoint"] = comparable.Differences{
				SourceValue:      r.Endpoint.ConnectionEndpoint,
				DestinationValue: otherResource.Endpoint.ConnectionEndpoint,
			}
		}

		if r.Endpoint.SecurityMode != otherResource.Endpoint.SecurityMode {
			differences["Endpoint.SecurityMode"] = comparable.Differences{
				SourceValue:      r.Endpoint.SecurityMode,
				DestinationValue: otherResource.Endpoint.SecurityMode,
			}
		}

		if r.Endpoint.ServerCertThumbprint != otherResource.Endpoint.ServerCertThumbprint {
			differences["Endpoint.ServerCertThumbprint"] = comparable.Differences{
				SourceValue:      r.Endpoint.ServerCertThumbprint,
				DestinationValue: otherResource.Endpoint.ServerCertThumbprint,
			}
		}

		if r.Endpoint.ClientCertVariable != otherResource.Endpoint.ClientCertVariable {
			differences["Endpoint.ClientCertVariable"] = comparable.Differences{
				SourceValue:      r.Endpoint.ClientCertVariable,
				DestinationValue: otherResource.Endpoint.ClientCertVariable,
			}
		}

		if r.Endpoint.CertificateStoreLocation != otherResource.Endpoint.CertificateStoreLocation {
			differences["Endpoint.CertificateStoreLocation"] = comparable.Differences{
				SourceValue:      r.Endpoint.CertificateStoreLocation,
				DestinationValue: otherResource.Endpoint.CertificateStoreLocation,
			}
		}

		if r.Endpoint.CertificateStoreName != otherResource.Endpoint.CertificateStoreName {
			differences["Endpoint.CertificateStoreName"] = comparable.Differences{
				SourceValue:      r.Endpoint.CertificateStoreName,
				DestinationValue: otherResource.Endpoint.CertificateStoreName,
			}
		}

		if r.Endpoint.AadCredentialType != otherResource.Endpoint.AadCredentialType {
			differences["Endpoint.AadCredentialType"] = comparable.Differences{
				SourceValue:      r.Endpoint.AadCredentialType,
				DestinationValue: otherResource.Endpoint.AadCredentialType,
			}
		}

		if r.Endpoint.AadClientCredentialSecret != otherResource.Endpoint.AadClientCredentialSecret {
			differences["Endpoint.AadClientCredentialSecret"] = comparable.Differences{
				SourceValue:      r.Endpoint.AadClientCredentialSecret,
				DestinationValue: otherResource.Endpoint.AadClientCredentialSecret,
			}
		}

		if r.Endpoint.AadUserCredentialUsername != otherResource.Endpoint.AadUserCredentialUsername {
			differences["Endpoint.AadUserCredentialUsername"] = comparable.Differences{
				SourceValue:      r.Endpoint.AadUserCredentialUsername,
				DestinationValue: otherResource.Endpoint.AadUserCredentialUsername,
			}
		}

		if r.Endpoint.AadUserCredentialPassword.HasValue != otherResource.Endpoint.AadUserCredentialPassword.HasValue {
			differences["Endpoint.AadUserCredentialPassword"] = comparable.Differences{
				SourceValue:      strconv.FormatBool(r.Endpoint.AadUserCredentialPassword.HasValue),
				DestinationValue: strconv.FormatBool(otherResource.Endpoint.AadUserCredentialPassword.HasValue),
			}
		}

		if r.Endpoint.DefaultWorkerPoolId != otherResource.Endpoint.DefaultWorkerPoolId {
			differences["Endpoint.DefaultWorkerPoolId"] = comparable.Differences{
				SourceValue:      r.Endpoint.DefaultWorkerPoolId,
				DestinationValue: otherResource.Endpoint.DefaultWorkerPoolId,
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

func (r *AzureServiceFabricResource) GetChildResources() []comparable.OctopusResource {
	return []comparable.OctopusResource{}
}

func (r *AzureServiceFabricResource) GetName() string {
	return r.Name
}

type AzureServiceFabricResource struct {
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
