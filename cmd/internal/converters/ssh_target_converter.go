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

const octopusdeploySshConnectionDeploymentTargetDataType = "octopusdeploy_deployment_targets"
const octopusdeploySshConnectionDeploymentTargetResourceType = "octopusdeploy_ssh_connection_deployment_target"

type SshTargetConverter struct {
	TargetConverter

	MachinePolicyConverter   ConverterWithStatelessById
	AccountConverter         ConverterAndLookupWithStatelessById
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

func (c SshTargetConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c SshTargetConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c SshTargetConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTargets {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.SshEndpointResource]{
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

		zap.L().Info("SSH Target: " + resource.Id + " " + resource.Name)
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c SshTargetConverter) validTarget(item octopus.SshEndpointResource) (bool, error) {
	err, noEnvironments := c.HasNoEnvironments(item)

	if err != nil {
		return false, err
	}

	if noEnvironments {
		return false, nil
	}

	return c.isSsh(item), nil
}

func (c SshTargetConverter) isSsh(resource octopus.SshEndpointResource) bool {
	return resource.Endpoint.CommunicationStyle == "Ssh"
}

func (c SshTargetConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c SshTargetConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c SshTargetConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTargets {
		return nil
	}

	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.SshEndpointResource{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.SshEndpointResource: %w", err)
	}

	if !c.isSsh(resource) {
		return nil
	}

	err, noEnvironments := c.HasNoEnvironments(resource)

	if err != nil {
		return err
	}

	if noEnvironments {
		return nil
	}

	zap.L().Info("SSH Target: " + resource.Id + " " + resource.Name)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c SshTargetConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTargets {
		return nil
	}

	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.SshEndpointResource{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.SshEndpointResource: %w", err)
	}

	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if !c.isSsh(resource) {
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
	thisResource.Lookup = "${data." + octopusdeploySshConnectionDeploymentTargetDataType + "." + resourceName + ".deployment_targets[0].id}"
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

func (c SshTargetConverter) buildData(resourceName string, resource octopus.SshEndpointResource) terraform.TerraformDeploymentTargetsData {
	return terraform.TerraformDeploymentTargetsData{
		Type:        octopusdeploySshConnectionDeploymentTargetDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c SshTargetConverter) writeData(file *hclwrite.File, resource octopus.SshEndpointResource, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

// toBashImport creates a bash script to import the resource
func (c SshTargetConverter) toBashImport(resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
    echo "jq is required" >&2
    exit 1
fi

if ! command -v curl &> /dev/null
then
    echo "curl is required" >&2
    exit 1
fi

RESOURCE_NAME="%s"
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Machines" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z "${RESOURCE_ID}" ]]
then
	echo "No target found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing target ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, octopusdeploySshConnectionDeploymentTargetDataType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c SshTargetConverter) toPowershellImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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
	Write-Error "No target found with the name $ResourceName"
	exit 1
}

echo "Importing target $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, octopusdeploySshConnectionDeploymentTargetDataType, resourceName), nil
		},
	})
}

func (c SshTargetConverter) toHcl(target octopus.SshEndpointResource, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(target.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + target.Id)
		return nil
	}

	if !c.isSsh(target) {
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
				return nil
			}
		} else {
			if err := c.exportDependencies(target, dependencies); err != nil {
				return nil
			}
		}
	}

	targetName := "target_" + sanitizer.SanitizeName(target.Name)

	if c.GenerateImportScripts && !stateless {
		c.toBashImport(targetName, target.Name, dependencies)
		c.toPowershellImport(targetName, target.Name, dependencies)
	}

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + targetName + ".tf"
	thisResource.Id = target.Id
	thisResource.Name = target.Name
	thisResource.ResourceType = c.GetResourceType()

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeploySshConnectionDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 " +
			"? data." + octopusdeploySshConnectionDeploymentTargetDataType + "." + targetName + ".deployment_targets[0].id " +
			": " + octopusdeploySshConnectionDeploymentTargetResourceType + "." + targetName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeploySshConnectionDeploymentTargetResourceType + "." + targetName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeploySshConnectionDeploymentTargetResourceType + "." + targetName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformSshConnectionDeploymentTarget{
			Id:                 strutil.InputPointerIfEnabled(c.IncludeIds, &target.Id),
			SpaceId:            strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", target.SpaceId)),
			Type:               octopusdeploySshConnectionDeploymentTargetResourceType,
			Name:               targetName,
			AccountId:          c.getAccount(target.Endpoint.AccountId, dependencies),
			Environments:       c.lookupEnvironments(target.EnvironmentIds, dependencies),
			Fingerprint:        target.Endpoint.Fingerprint,
			Host:               target.Endpoint.Host,
			ResourceName:       target.Name,
			Roles:              target.Roles,
			DotNetCorePlatform: &target.Endpoint.DotNetCorePlatform,
			MachinePolicyId:    c.getMachinePolicy(target.MachinePolicyId, dependencies),
			TenantTags:         c.Excluder.FilteredTenantTags(target.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, target, targetName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeploySshConnectionDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 ? 0 : 1}")
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

func (c SshTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c SshTargetConverter) lookupEnvironments(envs []string, dependencies *data.ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return lo.Filter(newEnvs, func(item string, index int) bool {
		return strings.TrimSpace(item) != ""
	})
}

func (c SshTargetConverter) getMachinePolicy(machine string, dependencies *data.ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c SshTargetConverter) getAccount(account string, dependencies *data.ResourceDetailsCollection) string {
	accountLookup := dependencies.GetResource("Accounts", account)
	if accountLookup == "" {
		return ""
	}

	return accountLookup
}

func (c SshTargetConverter) exportDependencies(target octopus.SshEndpointResource, dependencies *data.ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := c.MachinePolicyConverter.ToHclById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the accounts
	err = c.AccountConverter.ToHclById(target.Endpoint.AccountId, dependencies)

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

func (c SshTargetConverter) exportStatelessDependencies(target octopus.SshEndpointResource, dependencies *data.ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := c.MachinePolicyConverter.ToHclStatelessById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the accounts
	err = c.AccountConverter.ToHclStatelessById(target.Endpoint.AccountId, dependencies)

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
