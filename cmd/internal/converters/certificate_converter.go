package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const octopusdeployCertificateDataType = "octopusdeploy_certificates"
const octopusdeployCertificateResourceType = "octopusdeploy_certificate"

type CertificateConverter struct {
	Client                    client.OctopusClient
	DummySecretVariableValues bool
	DummySecretGenerator      dummy.DummySecretGenerator
	ExcludeTenantTags         args.StringSliceArgs
	ExcludeTenantTagSets      args.StringSliceArgs
	Excluder                  ExcludeByName
	TagSetConverter           ConvertToHclByResource[octopus.TagSet]
	ErrGroup                  *errgroup.Group
	ExcludeCertificates       args.StringSliceArgs
	ExcludeCertificatesRegex  args.StringSliceArgs
	ExcludeCertificatesExcept args.StringSliceArgs
	ExcludeAllCertificates    bool
	LimitResourceCount        int
	IncludeIds                bool
	IncludeSpaceInPopulation  bool
	GenerateImportScripts     bool
}

func (c CertificateConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c CertificateConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c CertificateConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllCertificates {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.Certificate]{
		Client: c.Client,
	}

	done := make(chan struct{})
	defer close(done)

	channel := batchClient.GetAllResourcesBatch(done, c.GetResourceType())

	for resourceWrapper := range channel {
		if resourceWrapper.Err != nil {
			return resourceWrapper.Err
		}

		resource := resourceWrapper.Res
		if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllCertificates, c.ExcludeCertificates, c.ExcludeCertificatesRegex, c.ExcludeCertificatesExcept) {
			continue
		}

		zap.L().Info("Certificate: " + resource.Id)
		err := c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c CertificateConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c CertificateConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c CertificateConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Certificate{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllCertificates, c.ExcludeCertificates, c.ExcludeCertificatesRegex, c.ExcludeCertificatesExcept) {
		return nil
	}

	zap.L().Info("Certificate: " + resource.Id)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c CertificateConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	certificate := octopus.Certificate{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &certificate)

	if err != nil {
		return err
	}

	if c.Excluder.IsResourceExcludedWithRegex(certificate.Name, c.ExcludeAllCertificates, c.ExcludeCertificates, c.ExcludeCertificatesRegex, c.ExcludeCertificatesExcept) {
		return nil
	}

	thisResource := data.ResourceDetails{}

	certificateName := "certificate_" + sanitizer.SanitizeName(certificate.Name)

	thisResource.FileName = "space_population/" + certificateName + ".tf"
	thisResource.Id = certificate.Id
	thisResource.Name = certificate.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployCertificateDataType + "." + certificateName + ".certificates[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(certificateName, certificate)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a certificate called \""+certificate.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.certificates) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c CertificateConverter) buildData(resourceName string, resource octopus.Certificate) terraform.TerraformCertificateData {
	return terraform.TerraformCertificateData{
		Type:        octopusdeployCertificateDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c CertificateConverter) writeData(file *hclwrite.File, resource octopus.Certificate, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

// toBashImport creates a bash script to import the resource
func (c CertificateConverter) toBashImport(resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
	dependencies.AddResource(data.ResourceDetails{
		FileName: "space_population/import_" + resourceName + ".sh",
		ToHcl: func() (string, error) {
			return fmt.Sprintf(`#!/bin/bash

# This script is used to import an exiting resource into the Terraform state.
# It is useful when importing a Terraform module into an Octopus space that
# already has existing resources.

# Make the script executable with the command:
# chmod +x ./import_%s.sh

# Alternativly, run the script with bash directly:
# /bin/bash ./import_%s.sh <options>

# Run "terraform init" to download any required providers and to configure the
# backend configuration

# Then run the import script. Replace the API key, instance URL, and Space ID 
# in the example below with the values of the space that the Terraform module 
# will be imported into.

# ./import_%s.sh API-xxxxxxxxxxxx https://yourinstance.octopus.app Spaces-1234

if [[ $# -ne 3 ]]
then
	echo "Usage: ./import_%s.sh <API Key> <Octopus URL> <Space ID>"
    echo "Example: ./import_%s.sh API-xxxxxxxxxxxx https://yourinstance.octopus.app Spaces-1234"
	exit 1
fi

if ! command -v jq &> /dev/null
then
    echo "jq is required"
    exit 1
fi

if ! command -v curl &> /dev/null
then
    echo "curl is required"
    exit 1
fi

RESOURCE_NAME="%s"
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Certificates" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No certificate found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing certificate ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, octopusdeployCertificateResourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c *CertificateConverter) toPowershellImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
	dependencies.AddResource(data.ResourceDetails{
		FileName: "space_population/import_" + resourceName + ".ps1",
		ToHcl: func() (string, error) {
			return fmt.Sprintf(`# This script is used to import an exiting resource into the Terraform state.
# It is useful when importing a Terraform module into an Octopus space that
# already has existing resources.

# Run "terraform init" to download any required providers and to configure the
# backend configuration

# Then run the import script. Replace the API key, instance URL, and Space ID 
# in the example below with the values of the space that the Terraform module 
# will be imported into.

# ./import_%s.ps1 API-xxxxxxxxxxxx https://yourinstance.octopus.app Spaces-1234

param (
    [Parameter(Mandatory=$true)]
    [string]$ApiKey,

    [Parameter(Mandatory=$true)]
    [string]$Url,

    [Parameter(Mandatory=$true)]
    [string]$SpaceId
)

$ResourceName="%s"

$headers = @{
    "X-Octopus-ApiKey" = $ApiKey
}

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Certificates?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No certificate found with the name $ResourceName"
	exit 1
}

echo "Importing certificate $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, octopusdeployCertificateResourceType, resourceName), nil
		},
	})
}

func (c CertificateConverter) toHcl(certificate octopus.Certificate, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.Excluder.IsResourceExcludedWithRegex(certificate.Name, c.ExcludeAllCertificates, c.ExcludeCertificates, c.ExcludeCertificatesRegex, c.ExcludeCertificatesExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + certificate.Id)
		return nil
	}

	/*
		Note we don't export the tenants or environments that this certificate might be exposed to.
		It is assumed the exported project links up all required environments, and the certificate
		will link itself to any available environments or tenants.
	*/

	certificateName := "certificate_" + sanitizer.SanitizeName(certificate.Name)

	if c.GenerateImportScripts {
		c.toBashImport(certificateName, certificate.Name, dependencies)
		c.toPowershellImport(certificateName, certificate.Name, dependencies)
	}

	thisResource := data.ResourceDetails{}
	thisResource.Name = certificate.Name
	thisResource.FileName = "space_population/" + certificateName + ".tf"
	thisResource.Id = certificate.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${" + octopusdeployCertificateResourceType + "." + certificateName + ".id}"
	thisResource.Parameters = []data.ResourceParameter{
		{
			Label:         "Certificate " + certificate.Name + " password",
			Description:   "The password associated with the certificate \"" + certificate.Name + "\"",
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, certificate.Name, "Password"),
			ParameterType: "Password",
			Sensitive:     true,
			VariableName:  certificateName + "_password",
		},
		{
			Label:         "Certificate " + certificate.Name + " contents",
			Description:   "The content of the certificate \"" + certificate.Name + "\"",
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, certificate.Name, "Data"),
			ParameterType: "Data",
			Sensitive:     true,
			VariableName:  certificateName + "_data",
		},
	}

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployCertificateDataType + "." + certificateName + ".certificates) != 0 " +
			"? data." + octopusdeployCertificateDataType + "." + certificateName + ".certificates[0].id " +
			": " + octopusdeployCertificateResourceType + "." + certificateName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployCertificateResourceType + "." + certificateName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployCertificateResourceType + "." + certificateName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		err := c.writeMainResource(file, certificateName, certificate, recursive, stateless, dependencies)

		if err != nil {
			return "", err
		}

		err = c.writeVariables(file, certificateName, certificate, dependencies)

		if err != nil {
			return "", err
		}

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c CertificateConverter) writeVariables(file *hclwrite.File, certificateName string, certificate octopus.Certificate, dependencies *data.ResourceDetailsCollection) error {

	defaultPassword := ""
	certificatePassword := terraform.TerraformVariable{
		Name:        certificateName + "_password",
		Type:        "string",
		Nullable:    true,
		Sensitive:   true,
		Description: "The password used by the certificate " + certificate.Name,
		Default:     &defaultPassword,
	}

	if c.DummySecretVariableValues {
		certificatePassword.Default = c.DummySecretGenerator.GetDummyCertificatePassword()
		dependencies.AddDummy(data.DummyVariableReference{
			VariableName: certificateName + "_password",
			ResourceName: certificate.Name,
			ResourceType: c.GetResourceType(),
		})
	}

	block := gohcl.EncodeAsBlock(certificatePassword, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	certificateData := terraform.TerraformVariable{
		Name:        certificateName + "_data",
		Type:        "string",
		Nullable:    false,
		Sensitive:   true,
		Description: "The certificate data used by the certificate " + certificate.Name,
	}

	if c.DummySecretVariableValues {
		certificateData.Default = c.DummySecretGenerator.GetDummyCertificate()
		dependencies.AddDummy(data.DummyVariableReference{
			VariableName: certificateName + "_data",
			ResourceName: certificate.Name,
			ResourceType: c.GetResourceType(),
		})
	}

	block = gohcl.EncodeAsBlock(certificateData, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	return nil
}

func (c CertificateConverter) writeMainResource(file *hclwrite.File, certificateName string, certificate octopus.Certificate, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	terraformResource := terraform.TerraformCertificate{
		Id:              strutil.InputPointerIfEnabled(c.IncludeIds, &certificate.Id),
		Type:            octopusdeployCertificateResourceType,
		Name:            certificateName,
		SpaceId:         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", certificate.SpaceId)),
		ResourceName:    certificate.Name,
		Password:        "${var." + certificateName + "_password}",
		CertificateData: "${var." + certificateName + "_data}",
		Archived:        &certificate.Archived,
		//CertificateDataFormat:           certificate.CertificateDataFormat,
		Environments: c.lookupEnvironments(certificate.EnvironmentIds, dependencies),
		//HasPrivateKey:                   certificate.HasPrivateKey,
		//IsExpired:                       certificate.IsExpired,
		//IssuerCommonName:                certificate.IssuerCommonName,
		//IssuerDistinguishedName:         certificate.IssuerDistinguishedName,
		//IssuerOrganization:              certificate.IssuerOrganization,
		//NotAfter:                        certificate.NotAfter,
		//NotBefore:                       certificate.NotBefore,
		Notes: &certificate.Notes,
		//ReplacedBy:                      nil, // ReplacedBy does not seem to be used
		//SelfSigned:                      certificate.SelfSigned,
		//SerialNumber:                    certificate.SerialNumber,
		//SignatureAlgorithmName:          certificate.SignatureAlgorithmName,
		//SubjectAlternativeNames:         certificate.SubjectAlternativeNames,
		//SubjectCommonName:               certificate.SubjectCommonName,
		//SubjectDistinguishedName:        certificate.SubjectDistinguishedName,
		//SubjectOrganization:             certificate.SubjectOrganization,
		TenantTags:                      c.Excluder.FilteredTenantTags(certificate.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
		TenantedDeploymentParticipation: &certificate.TenantedDeploymentParticipation,
		Tenants:                         c.lookupTenants(certificate.TenantIds, dependencies),
		//Thumbprint:                      certificate.Thumbprint,
		//Version:                         certificate.Version,
	}

	if stateless {
		c.writeData(file, certificate, certificateName)
		terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployCertificateDataType + "." + certificateName + ".certificates) != 0 ? 0 : 1}")
	}

	targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")
	err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, targetBlock, dependencies, recursive)

	// When using dummy values, we expect the secrets will be updated later
	if c.DummySecretVariableValues || stateless {

		ignoreAll := terraform.EmptyBlock{}
		lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
		targetBlock.Body().AppendBlock(lifecycleBlock)

		if c.DummySecretVariableValues {
			hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "[password, certificate_data]")
		}

		if stateless {
			hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
		}
	}

	if err != nil {
		return err
	}

	file.Body().AppendBlock(targetBlock)

	return nil
}

func (c CertificateConverter) GetResourceType() string {
	return "Certificates"
}

func (c CertificateConverter) lookupEnvironments(envs []string, dependencies *data.ResourceDetailsCollection) []string {
	newEnvs := make([]string, 0)
	for _, v := range envs {
		environment := dependencies.GetResource("Environments", v)
		if environment != "" {
			newEnvs = append(newEnvs, environment)
		}
	}
	return newEnvs
}

func (c CertificateConverter) lookupTenants(tenants []string, dependencies *data.ResourceDetailsCollection) []string {
	newTenants := make([]string, 0)
	for _, v := range tenants {
		tenant := dependencies.GetResource("Tenants", v)
		if tenant != "" {
			newTenants = append(newTenants, tenant)
		}
	}
	return newTenants
}
