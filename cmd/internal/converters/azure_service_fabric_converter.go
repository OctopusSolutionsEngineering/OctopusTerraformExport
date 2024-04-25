package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"strings"
)

const octopusdeployAzureServiceFabricClusterDeploymentDataType = "octopusdeploy_deployment_targets"
const octopusdeployAzureServiceFabricClusterDeploymentResourceType = "octopusdeploy_azure_service_fabric_cluster_deployment_target"

type AzureServiceFabricTargetConverter struct {
	TargetConverter

	MachinePolicyConverter    ConverterWithStatelessById
	EnvironmentConverter      ConverterAndLookupWithStatelessById
	ExcludeAllTargets         bool
	ExcludeTargets            args.StringSliceArgs
	ExcludeTargetsRegex       args.StringSliceArgs
	ExcludeTargetsExcept      args.StringSliceArgs
	DummySecretVariableValues bool
	DummySecretGenerator      DummySecretGenerator
	ExcludeTenantTags         args.StringSliceArgs
	ExcludeTenantTagSets      args.StringSliceArgs
	TagSetConverter           ConvertToHclByResource[octopus.TagSet]
	ErrGroup                  *errgroup.Group
	IncludeIds                bool
	LimitResourceCount        int
	IncludeSpaceInPopulation  bool
	GenerateImportScripts     bool
}

func (c AzureServiceFabricTargetConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c AzureServiceFabricTargetConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c AzureServiceFabricTargetConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTargets {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.AzureServiceFabricResource]{
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

		valid, err := c.validTarget(resource)

		if err != nil {
			return err
		}

		if !valid {
			continue
		}

		zap.L().Info("Azure Service Fabric Target: " + resource.Id)
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c AzureServiceFabricTargetConverter) validTarget(item octopus.AzureServiceFabricResource) (bool, error) {
	err, noEnvironments := c.HasNoEnvironments(item)

	if err != nil {
		return false, err
	}

	if noEnvironments {
		return false, nil
	}

	return c.isAzureServiceFabricCluster(item), nil
}

func (c AzureServiceFabricTargetConverter) isAzureServiceFabricCluster(resource octopus.AzureServiceFabricResource) bool {
	return resource.Endpoint.CommunicationStyle == "AzureServiceFabricCluster"
}

func (c AzureServiceFabricTargetConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c AzureServiceFabricTargetConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c AzureServiceFabricTargetConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.AzureServiceFabricResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if !c.isAzureServiceFabricCluster(resource) {
		return nil
	}

	zap.L().Info("Azure Service Fabric Target: " + resource.Id)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c AzureServiceFabricTargetConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.AzureServiceFabricResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if !c.isAzureServiceFabricCluster(resource) {
		return nil
	}

	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceName := "target_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Name = resource.Name
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployAzureServiceFabricClusterDeploymentDataType + "." + resourceName + ".deployment_targets[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(resourceName, resource)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a deployment target called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.deployment_targets) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c AzureServiceFabricTargetConverter) buildData(resourceName string, resource octopus.AzureServiceFabricResource) terraform.TerraformDeploymentTargetsData {
	return terraform.TerraformDeploymentTargetsData{
		Type:        octopusdeployAzureServiceFabricClusterDeploymentDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c AzureServiceFabricTargetConverter) writeData(file *hclwrite.File, resource octopus.AzureServiceFabricResource, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

// toBashImport creates a bash script to import the resource
func (c AzureServiceFabricTargetConverter) toBashImport(resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Machines" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No target found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing target ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, octopusdeployAzureServiceFabricClusterDeploymentResourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c *AzureServiceFabricTargetConverter) toPowershellImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Machines?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No target found with the name $ResourceName"
	exit 1
}

echo "Importing target $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, octopusdeployAzureServiceFabricClusterDeploymentResourceType, resourceName), nil
		},
	})
}

func (c AzureServiceFabricTargetConverter) toHcl(target octopus.AzureServiceFabricResource, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(target.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + target.Id)
		return nil
	}

	if !c.isAzureServiceFabricCluster(target) {
		return nil
	}

	if recursive {
		if stateless {
			if err := c.exportStatelessDependencies(target, dependencies); err != nil {
				return err
			}
		} else {
			if err := c.exportDependencies(target, dependencies); err != nil {
				return err
			}
		}
	}

	targetName := "target_" + sanitizer.SanitizeName(target.Name)

	if c.GenerateImportScripts {
		c.toBashImport(targetName, target.Name, dependencies)
		c.toPowershellImport(targetName, target.Name, dependencies)
	}

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + targetName + ".tf"
	thisResource.Id = target.Id
	thisResource.Name = target.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${" + octopusdeployAzureServiceFabricClusterDeploymentResourceType + "." + targetName + ".id}"
	thisResource.Parameters = []data.ResourceParameter{
		{
			VariableName:  targetName,
			Label:         "Service Fabric Target " + target.Name + " aad_user_credential_password",
			Description:   "The aad_user_credential_password value associated with the target \"" + target.Name + "\"",
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, target.Name, "AadUserCredentialPassword"),
			ParameterType: "AadUserCredentialPassword",
			Sensitive:     true,
		},
	}

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployAzureServiceFabricClusterDeploymentDataType + "." + targetName + ".deployment_targets) != 0 " +
			"? data." + octopusdeployAzureServiceFabricClusterDeploymentDataType + "." + targetName + ".deployment_targets[0].id " +
			": " + octopusdeployAzureServiceFabricClusterDeploymentResourceType + "." + targetName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployAzureServiceFabricClusterDeploymentResourceType + "." + targetName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployAzureServiceFabricClusterDeploymentResourceType + "." + targetName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		passwordLookup := "${var." + targetName + "}"

		terraformResource := terraform.TerraformAzureServiceFabricClusterDeploymentTarget{
			Type:                            octopusdeployAzureServiceFabricClusterDeploymentResourceType,
			Name:                            targetName,
			Id:                              strutil.InputPointerIfEnabled(c.IncludeIds, &target.Id),
			SpaceId:                         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", target.SpaceId)),
			Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
			ResourceName:                    target.Name,
			Roles:                           target.Roles,
			ConnectionEndpoint:              target.Endpoint.ConnectionEndpoint,
			AadClientCredentialSecret:       &target.Endpoint.AadClientCredentialSecret,
			AadCredentialType:               &target.Endpoint.AadCredentialType,
			AadUserCredentialPassword:       &passwordLookup,
			AadUserCredentialUsername:       &target.Endpoint.AadUserCredentialUsername,
			CertificateStoreLocation:        &target.Endpoint.CertificateStoreLocation,
			CertificateStoreName:            &target.Endpoint.CertificateStoreName,
			ClientCertificateVariable:       &target.Endpoint.ClientCertVariable,
			HealthStatus:                    &target.HealthStatus,
			IsDisabled:                      &target.IsDisabled,
			MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId, dependencies),
			OperatingSystem:                 nil,
			SecurityMode:                    nil,
			ServerCertificateThumbprint:     nil,
			ShellName:                       &target.ShellName,
			ShellVersion:                    &target.ShellVersion,
			Status:                          nil,
			StatusSummary:                   nil,
			TenantTags:                      c.Excluder.FilteredTenantTags(target.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			TenantedDeploymentParticipation: &target.TenantedDeploymentParticipation,
			Tenants:                         dependencies.GetResources("Tenants", target.TenantIds...),
			Thumbprint:                      &target.Thumbprint,
			Uri:                             nil,
			Endpoint:                        nil,
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, target, targetName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployAzureServiceFabricClusterDeploymentDataType + "." + targetName + ".deployment_targets) != 0 ? 0 : 1}")
		}

		targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, targetBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		// When using dummy values, we expect the secrets will be updated later
		if c.DummySecretVariableValues || stateless {

			ignoreAll := terraform.EmptyBlock{}
			lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
			targetBlock.Body().AppendBlock(lifecycleBlock)

			if c.DummySecretVariableValues {
				hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "[aad_user_credential_password]")
			}

			if stateless {
				hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
			}
		}

		file.Body().AppendBlock(targetBlock)

		secretVariableResource := terraform.TerraformVariable{
			Name:        targetName,
			Type:        "string",
			Nullable:    true,
			Sensitive:   true,
			Description: "The aad_user_credential_password value associated with the target \"" + target.Name + "\"",
		}

		if c.DummySecretVariableValues {
			secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
		}

		block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
		hcl.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c AzureServiceFabricTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c AzureServiceFabricTargetConverter) lookupEnvironments(envs []string, dependencies *data.ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return lo.Filter(newEnvs, func(item string, index int) bool {
		return strings.TrimSpace(item) != ""
	})
}

func (c AzureServiceFabricTargetConverter) getMachinePolicy(machine string, dependencies *data.ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c AzureServiceFabricTargetConverter) getWorkerPool(pool string, dependencies *data.ResourceDetailsCollection) *string {
	if len(pool) == 0 {
		return nil
	}

	machineLookup := dependencies.GetResource("WorkerPools", pool)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c AzureServiceFabricTargetConverter) exportDependencies(target octopus.AzureServiceFabricResource, dependencies *data.ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := c.MachinePolicyConverter.ToHclById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the environments
	for _, e := range target.EnvironmentIds {
		err = c.EnvironmentConverter.ToHclById(e, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c AzureServiceFabricTargetConverter) exportStatelessDependencies(target octopus.AzureServiceFabricResource, dependencies *data.ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := c.MachinePolicyConverter.ToHclStatelessById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the environments
	for _, e := range target.EnvironmentIds {
		err = c.EnvironmentConverter.ToHclStatelessById(e, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}
