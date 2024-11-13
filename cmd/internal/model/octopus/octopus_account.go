package octopus

import (
	comparable2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/comparable"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"strconv"
)

type Account struct {
	Id                              string
	Name                            string
	Slug                            *string
	Description                     *string
	SpaceId                         string
	EnvironmentIds                  []string
	TenantedDeploymentParticipation *string
	TenantIds                       []string
	TenantTags                      []string
	AccountType                     string

	// token
	Token Secret

	// aws
	AccessKey *string
	SecretKey Secret

	// azure
	SubscriptionNumber                *string
	ClientId                          *string
	TenantId                          *string
	Password                          Secret
	AzureEnvironment                  *string
	ResourceManagementEndpointBaseUri *string
	ActiveDirectoryEndpointBaseUri    *string

	// azure subscription
	ServiceManagementEndpointBaseUri *string
	ServiceManagementEndpointSuffix  *string
	CertificateBytes                 Secret
	CertificateThumbprint            *string

	// username
	Username *string

	// google
	JsonKey Secret
}

type Secret struct {
	HasValue bool
	NewValue *string
	Hint     *string
}

func (r *Account) Compare(other comparable2.OctopusResource) comparable2.OctopusResourceComparison {
	differences := map[string]comparable2.Differences{}

	if otherResource, ok := other.(*Account); ok {
		if r.Name != otherResource.Name {
			differences["Name"] = comparable2.Differences{
				SourceValue:      r.Name,
				DestinationValue: otherResource.Name,
			}
		}

		if strutil.EmptyIfNil(r.Slug) != strutil.EmptyIfNil(otherResource.Slug) {
			differences["Slug"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.Slug),
				DestinationValue: strutil.EmptyIfNil(otherResource.Slug),
			}
		}

		if strutil.EmptyIfNil(r.Description) != strutil.EmptyIfNil(otherResource.Description) {
			differences["Description"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.Description),
				DestinationValue: strutil.EmptyIfNil(otherResource.Description),
			}
		}

		if strutil.EmptyIfNil(r.TenantedDeploymentParticipation) != strutil.EmptyIfNil(otherResource.TenantedDeploymentParticipation) {
			differences["TenantedDeploymentParticipation"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.TenantedDeploymentParticipation),
				DestinationValue: strutil.EmptyIfNil(otherResource.TenantedDeploymentParticipation),
			}
		}

		if r.AccountType != otherResource.AccountType {
			differences["AccountType"] = comparable2.Differences{
				SourceValue:      r.AccountType,
				DestinationValue: otherResource.AccountType,
			}
		}

		if strutil.EmptyIfNil(r.AccessKey) != strutil.EmptyIfNil(otherResource.AccessKey) {
			differences["AccessKey"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.AccessKey),
				DestinationValue: strutil.EmptyIfNil(otherResource.AccessKey),
			}
		}

		if r.SecretKey.HasValue != otherResource.SecretKey.HasValue {
			differences["SecretKey"] = comparable2.Differences{
				SourceValue:      strconv.FormatBool(r.SecretKey.HasValue),
				DestinationValue: strconv.FormatBool(otherResource.SecretKey.HasValue),
			}
		}

		if strutil.EmptyIfNil(r.SubscriptionNumber) != strutil.EmptyIfNil(otherResource.SubscriptionNumber) {
			differences["SubscriptionNumber"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.SubscriptionNumber),
				DestinationValue: strutil.EmptyIfNil(otherResource.SubscriptionNumber),
			}
		}

		if strutil.EmptyIfNil(r.ClientId) != strutil.EmptyIfNil(otherResource.ClientId) {
			differences["ClientId"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.ClientId),
				DestinationValue: strutil.EmptyIfNil(otherResource.ClientId),
			}
		}

		if strutil.EmptyIfNil(r.TenantId) != strutil.EmptyIfNil(otherResource.TenantId) {
			differences["TenantId"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.TenantId),
				DestinationValue: strutil.EmptyIfNil(otherResource.TenantId),
			}
		}

		if r.Password.HasValue != otherResource.Password.HasValue {
			differences["Password"] = comparable2.Differences{
				SourceValue:      strconv.FormatBool(r.Password.HasValue),
				DestinationValue: strconv.FormatBool(otherResource.Password.HasValue),
			}
		}

		if strutil.EmptyIfNil(r.AzureEnvironment) != strutil.EmptyIfNil(otherResource.AzureEnvironment) {
			differences["AzureEnvironment"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.AzureEnvironment),
				DestinationValue: strutil.EmptyIfNil(otherResource.AzureEnvironment),
			}
		}

		if strutil.EmptyIfNil(r.ResourceManagementEndpointBaseUri) != strutil.EmptyIfNil(otherResource.ResourceManagementEndpointBaseUri) {
			differences["ResourceManagementEndpointBaseUri"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.ResourceManagementEndpointBaseUri),
				DestinationValue: strutil.EmptyIfNil(otherResource.ResourceManagementEndpointBaseUri),
			}
		}

		if strutil.EmptyIfNil(r.ActiveDirectoryEndpointBaseUri) != strutil.EmptyIfNil(otherResource.ActiveDirectoryEndpointBaseUri) {
			differences["ActiveDirectoryEndpointBaseUri"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.ActiveDirectoryEndpointBaseUri),
				DestinationValue: strutil.EmptyIfNil(otherResource.ActiveDirectoryEndpointBaseUri),
			}
		}

		if strutil.EmptyIfNil(r.ServiceManagementEndpointBaseUri) != strutil.EmptyIfNil(otherResource.ServiceManagementEndpointBaseUri) {
			differences["ServiceManagementEndpointBaseUri"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.ServiceManagementEndpointBaseUri),
				DestinationValue: strutil.EmptyIfNil(otherResource.ServiceManagementEndpointBaseUri),
			}
		}

		if strutil.EmptyIfNil(r.ServiceManagementEndpointSuffix) != strutil.EmptyIfNil(otherResource.ServiceManagementEndpointSuffix) {
			differences["ServiceManagementEndpointSuffix"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.ServiceManagementEndpointSuffix),
				DestinationValue: strutil.EmptyIfNil(otherResource.ServiceManagementEndpointSuffix),
			}
		}

		if r.CertificateBytes.HasValue != otherResource.CertificateBytes.HasValue {
			differences["CertificateBytes"] = comparable2.Differences{
				SourceValue:      strconv.FormatBool(r.CertificateBytes.HasValue),
				DestinationValue: strconv.FormatBool(otherResource.CertificateBytes.HasValue),
			}
		}

		if strutil.EmptyIfNil(r.CertificateThumbprint) != strutil.EmptyIfNil(otherResource.CertificateThumbprint) {
			differences["CertificateThumbprint"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.CertificateThumbprint),
				DestinationValue: strutil.EmptyIfNil(otherResource.CertificateThumbprint),
			}
		}

		if strutil.EmptyIfNil(r.Username) != strutil.EmptyIfNil(otherResource.Username) {
			differences["Username"] = comparable2.Differences{
				SourceValue:      strutil.EmptyIfNil(r.Username),
				DestinationValue: strutil.EmptyIfNil(otherResource.Username),
			}
		}

		if r.JsonKey.HasValue != otherResource.JsonKey.HasValue {
			differences["JsonKey"] = comparable2.Differences{
				SourceValue:      strconv.FormatBool(r.JsonKey.HasValue),
				DestinationValue: strconv.FormatBool(otherResource.JsonKey.HasValue),
			}
		}
	}

	return comparable2.OctopusResourceComparison{
		SourceResource:                 r,
		DestinationResource:            other,
		Differences:                    differences,
		ChildOctopusResourceComparison: nil,
	}
}

func (r *Account) GetChildResources() []comparable2.OctopusResource {
	return []comparable2.OctopusResource{}
}

func (r *Account) GetName() string {
	return r.Name
}
