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

const octopusdeployPollingTentacleDeploymentTargetDataType = "octopusdeploy_deployment_targets"
const octopusdeployPollingTentacleDeploymentTargetResourceType = "octopusdeploy_polling_tentacle_deployment_target"

type PollingTargetConverter struct {
	TargetConverter

	MachinePolicyConverter   ConverterWithStatelessById
	EnvironmentConverter     ConverterAndLookupWithStatelessById
	ExcludeAllTargets        bool
	ExcludeTargets           args.StringSliceArgs
	ExcludeTargetsRegex      args.StringSliceArgs
	ExcludeTargetsExcept     args.StringSliceArgs
	ExcludeTenantTags        args.StringSliceArgs
	ExcludeTenantTagSets     args.StringSliceArgs
	TagSetConverter          ConvertToHclByResource[octopus.TagSet]
	ErrGroup                 *errgroup.Group
	IncludeIds               bool
	LimitResourceCount       int
	IncludeSpaceInPopulation bool
	GenerateImportScripts    bool
}

func (c PollingTargetConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c PollingTargetConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c PollingTargetConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTargets {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.PollingEndpointResource]{
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

		zap.L().Info("Polling Target: " + resource.Id + " " + resource.Name)
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c PollingTargetConverter) validTarget(item octopus.PollingEndpointResource) (bool, error) {
	err, noEnvironments := c.HasNoEnvironments(item)

	if err != nil {
		return false, err
	}

	if noEnvironments {
		return false, nil
	}

	return c.isPollingTarget(item), nil
}

func (c PollingTargetConverter) isPollingTarget(resource octopus.PollingEndpointResource) bool {
	return resource.Endpoint.CommunicationStyle == "TentacleActive"
}

func (c PollingTargetConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c PollingTargetConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c PollingTargetConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.PollingEndpointResource{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.PollingEndpointResource: %w", err)
	}

	if !c.isPollingTarget(resource) {
		return nil
	}

	err, noEnvironments := c.HasNoEnvironments(resource)

	if err != nil {
		return err
	}

	if noEnvironments {
		return nil
	}

	zap.L().Info("Polling Target: " + resource.Id + " " + resource.Name)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c PollingTargetConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.PollingEndpointResource{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.PollingEndpointResource: %w", err)
	}

	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if !c.isPollingTarget(resource) {
		return nil
	}

	err, noEnvironments := c.HasNoEnvironments(resource)

	if err != nil {
		return err
	}

	if noEnvironments {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceName := "target_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.Name = resource.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployPollingTentacleDeploymentTargetDataType + "." + resourceName + ".deployment_targets[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(resourceName, resource)
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c PollingTargetConverter) buildData(resourceName string, resource octopus.PollingEndpointResource) terraform.TerraformDeploymentTargetsData {
	return terraform.TerraformDeploymentTargetsData{
		Type:        octopusdeployPollingTentacleDeploymentTargetDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c PollingTargetConverter) writeData(file *hclwrite.File, resource octopus.PollingEndpointResource, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

// toBashImport creates a bash script to import the resource
func (c PollingTargetConverter) toBashImport(resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, octopusdeployPollingTentacleDeploymentTargetDataType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c PollingTargetConverter) toPowershellImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, octopusdeployPollingTentacleDeploymentTargetDataType, resourceName), nil
		},
	})
}

func (c PollingTargetConverter) toHcl(target octopus.PollingEndpointResource, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(target.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + target.Id)
		return nil
	}

	if !c.isPollingTarget(target) {
		return nil
	}

	err, noEnvironments := c.HasNoEnvironments(target)

	if err != nil {
		return err
	}

	if noEnvironments {
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

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployPollingTentacleDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 " +
			"? data." + octopusdeployPollingTentacleDeploymentTargetDataType + "." + targetName + ".deployment_targets[0].id " +
			": " + octopusdeployPollingTentacleDeploymentTargetResourceType + "." + targetName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployPollingTentacleDeploymentTargetResourceType + "." + targetName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployPollingTentacleDeploymentTargetResourceType + "." + targetName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformPollingTentacleDeploymentTarget{
			Id:                              strutil.InputPointerIfEnabled(c.IncludeIds, &target.Id),
			SpaceId:                         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", target.SpaceId)),
			Type:                            octopusdeployPollingTentacleDeploymentTargetResourceType,
			Name:                            targetName,
			Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
			ResourceName:                    target.Name,
			Roles:                           target.Roles,
			TentacleUrl:                     target.Endpoint.Uri,
			CertificateSignatureAlgorithm:   nil,
			HealthStatus:                    nil,
			IsDisabled:                      &target.IsDisabled,
			MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId, dependencies),
			OperatingSystem:                 nil,
			ShellName:                       &target.ShellName,
			ShellVersion:                    &target.ShellVersion,
			Status:                          nil,
			StatusSummary:                   nil,
			TenantTags:                      c.Excluder.FilteredTenantTags(target.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			TenantedDeploymentParticipation: &target.TenantedDeploymentParticipation,
			Tenants:                         dependencies.GetResources("Tenants", target.TenantIds...),
			TentacleVersionDetails:          terraform.TerraformTentacleVersionDetails{},
			Uri:                             nil,
			Thumbprint:                      target.Thumbprint,
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, target, targetName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployPollingTentacleDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 ? 0 : 1}")
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, block, dependencies, recursive)
		if err != nil {
			return "", err
		}
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c PollingTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c PollingTargetConverter) lookupEnvironments(envs []string, dependencies *data.ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return lo.Filter(newEnvs, func(item string, index int) bool {
		return strings.TrimSpace(item) != ""
	})
}

func (c PollingTargetConverter) getMachinePolicy(machine string, dependencies *data.ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c PollingTargetConverter) exportDependencies(target octopus.PollingEndpointResource, dependencies *data.ResourceDetailsCollection) error {

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

func (c PollingTargetConverter) exportStatelessDependencies(target octopus.PollingEndpointResource, dependencies *data.ResourceDetailsCollection) error {

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
