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
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const octopusdeployEnvironmentsDataType = "octopusdeploy_environments"
const octopusdeployEnvironmentsResourceType = "octopusdeploy_environment"

type EnvironmentConverter struct {
	Client                    client.OctopusClient
	ErrGroup                  *errgroup.Group
	ExcludeEnvironments       args.StringSliceArgs
	ExcludeEnvironmentsRegex  args.StringSliceArgs
	ExcludeEnvironmentsExcept args.StringSliceArgs
	ExcludeAllEnvironments    bool
	Excluder                  ExcludeByName
	IncludeIds                bool
	LimitResourceCount        int
	IncludeSpaceInPopulation  bool
	GenerateImportScripts     bool
}

func (c EnvironmentConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c EnvironmentConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c EnvironmentConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllEnvironments {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.Environment]{
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
		if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllEnvironments, c.ExcludeEnvironments, c.ExcludeEnvironmentsRegex, c.ExcludeEnvironmentsExcept) {
			continue
		}

		zap.L().Info("Environment: " + resource.Id)
		err := c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}
func (c EnvironmentConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c EnvironmentConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c EnvironmentConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllEnvironments {
		return nil
	}

	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Environment{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Environment: " + resource.Id)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c EnvironmentConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	environment := octopus.Environment{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &environment)

	if err != nil {
		return err
	}

	if c.Excluder.IsResourceExcludedWithRegex(environment.Name, c.ExcludeAllEnvironments, c.ExcludeEnvironments, c.ExcludeEnvironmentsRegex, c.ExcludeEnvironmentsExcept) {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceName := "environment_" + sanitizer.SanitizeName(environment.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = environment.Id
	thisResource.Name = environment.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployEnvironmentsDataType + "." + resourceName + ".environments[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(resourceName, environment)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve an environment called \""+environment.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.environments) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c EnvironmentConverter) buildData(resourceName string, resource octopus.Environment) terraform.TerraformEnvironmentData {
	return terraform.TerraformEnvironmentData{
		Type:        octopusdeployEnvironmentsDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c EnvironmentConverter) writeData(file *hclwrite.File, resource octopus.Environment, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c EnvironmentConverter) getLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data." + octopusdeployEnvironmentsDataType + "." + resourceName + ".environments) != 0 " +
			"? data." + octopusdeployEnvironmentsDataType + "." + resourceName + ".environments[0].id " +
			": " + octopusdeployEnvironmentsResourceType + "." + resourceName + "[0].id}"
	}
	return "${" + octopusdeployEnvironmentsResourceType + "." + resourceName + ".id}"

}

func (c EnvironmentConverter) getDependency(stateless bool, resourceName string) string {
	if stateless {
		return "${" + octopusdeployEnvironmentsResourceType + "." + resourceName + "}"
	}

	return ""
}

func (c EnvironmentConverter) getCount(stateless bool, resourceName string) *string {
	if stateless {
		return strutil.StrPointer("${length(data." + octopusdeployEnvironmentsDataType + "." + resourceName + ".environments) != 0 ? 0 : 1}")
	}

	return nil
}

// toBashImport creates a bash script to import the resource
func (c EnvironmentConverter) toBashImport(resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Environments" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No environment found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing environment ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, octopusdeployEnvironmentsResourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c EnvironmentConverter) toPowershellImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Environments?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No environment found with the name $ResourceName"
	exit 1
}

echo "Importing environment $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, octopusdeployEnvironmentsResourceType, resourceName), nil
		},
	})
}

func (c EnvironmentConverter) toHcl(environment octopus.Environment, _ bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.Excluder.IsResourceExcludedWithRegex(environment.Name, c.ExcludeAllEnvironments, c.ExcludeEnvironments, c.ExcludeEnvironmentsRegex, c.ExcludeEnvironmentsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + environment.Id)
		return nil
	}

	resourceName := "environment_" + sanitizer.SanitizeName(environment.Name)

	if c.GenerateImportScripts {
		c.toBashImport(resourceName, environment.Name, dependencies)
		c.toPowershellImport(resourceName, environment.Name, dependencies)
	}

	thisResource := data.ResourceDetails{}
	thisResource.Name = environment.Name
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = environment.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = c.getLookup(stateless, resourceName)
	thisResource.Dependency = c.getDependency(stateless, resourceName)

	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		terraformResource := terraform.TerraformEnvironment{
			Type:                       octopusdeployEnvironmentsResourceType,
			Name:                       resourceName,
			Id:                         strutil.InputPointerIfEnabled(c.IncludeIds, &environment.Id),
			Count:                      c.getCount(stateless, resourceName),
			SpaceId:                    strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", environment.SpaceId)),
			ResourceName:               environment.Name,
			Description:                environment.Description,
			AllowDynamicInfrastructure: environment.AllowDynamicInfrastructure,
			UseGuidedFailure:           environment.UseGuidedFailure,
			SortOrder:                  environment.SortOrder,
			JiraExtensionSettings: &terraform.TerraformJiraExtensionSettings{
				EnvironmentType: c.getJiraExtensionSettings(environment),
			},
			JiraServiceManagementExtensionSettings: &terraform.TerraformJiraServiceManagementExtensionSettings{
				IsEnabled: c.getJiraServiceManagementExtensionSettings(environment),
			},
			ServicenowExtensionSettings: &terraform.TerraformServicenowExtensionSettings{
				IsEnabled: c.getServiceNowChangeControlled(environment),
			},
		}

		if stateless {
			c.writeData(file, environment, resourceName)
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c EnvironmentConverter) getServiceNowChangeControlled(env octopus.Environment) bool {
	for _, setting := range env.ExtensionSettings {
		if setting.ExtensionId == "servicenow-integration" {
			v, ok := setting.Values["ServiceNowChangeControlled"]
			if ok {
				switch t := v.(type) {
				case bool:
					return t
				}
			}

		}
	}

	return false
}

func (c EnvironmentConverter) getJiraServiceManagementExtensionSettings(env octopus.Environment) bool {
	for _, setting := range env.ExtensionSettings {
		if setting.ExtensionId == "jiraservicemanagement-integration" {
			v, ok := setting.Values["JsmChangeControlled"]
			if ok {
				switch t := v.(type) {
				case bool:
					return t
				}
			}

		}
	}

	return false
}

func (c EnvironmentConverter) getJiraExtensionSettings(env octopus.Environment) string {
	for _, setting := range env.ExtensionSettings {
		if setting.ExtensionId == "jira-integration" {
			v, ok := setting.Values["JiraEnvironmentType"]
			if ok {
				switch t := v.(type) {
				case string:
					return t
				}
			}

		}
	}

	return "unmapped"
}

func (c EnvironmentConverter) GetResourceType() string {
	return "Environments"
}
