package converters

import (
	"fmt"
	"strings"

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

const octopusdeployParentEnvironmentsDataType = "octopusdeploy_parent_environments"
const octopusdeployParentEnvironmentResourceType = "octopusdeploy_parent_environment"

type ParentEnvironmentConverter struct {
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

func (c ParentEnvironmentConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c ParentEnvironmentConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c ParentEnvironmentConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllEnvironments {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.ParentEnvironment]{
		Client:     c.Client,
		ApiVersion: "v2",
	}

	done := make(chan struct{})
	defer close(done)

	channel := batchClient.GetAllResourcesBatchWithQueryParams(done, "environments", []string{"type", "Parent"})

	for resourceWrapper := range channel {
		if resourceWrapper.Err != nil {
			return resourceWrapper.Err
		}

		listItem := resourceWrapper.Res
		if c.Excluder.IsResourceExcludedWithRegex(listItem.Name, c.ExcludeAllEnvironments, c.ExcludeEnvironments, c.ExcludeEnvironmentsRegex, c.ExcludeEnvironmentsExcept) {
			continue
		}

		// The list endpoint returns incomplete details (notably the AutomaticDeprovisioningRule field is missing), so we fetch the full resource
		resource := octopus.ParentEnvironment{}
		_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), listItem.Id, &resource)

		if err != nil {
			return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.ParentEnvironment: %w", err)
		}

		zap.L().Info("Parent Environment: " + resource.Id + " " + resource.Name)
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c ParentEnvironmentConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c ParentEnvironmentConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c ParentEnvironmentConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllEnvironments {
		return nil
	}

	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.ParentEnvironment{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.ParentEnvironment: %w", err)
	}

	zap.L().Info("Parent Environment: " + resource.Id + " " + resource.Name)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c ParentEnvironmentConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	environment := octopus.ParentEnvironment{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &environment)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.ParentEnvironment: %w", err)
	}

	if c.Excluder.IsResourceExcludedWithRegex(environment.Name, c.ExcludeAllEnvironments, c.ExcludeEnvironments, c.ExcludeEnvironmentsRegex, c.ExcludeEnvironmentsExcept) {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceName := "parent_environment_" + sanitizer.SanitizeName(environment.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = environment.Id
	thisResource.Name = environment.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployParentEnvironmentsDataType + "." + resourceName + ".parent_environments[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(resourceName, environment)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a parent environment called \""+environment.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.environments) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c ParentEnvironmentConverter) buildData(resourceName string, resource octopus.ParentEnvironment) terraform.TerraformParentEnvironmentData {
	return terraform.TerraformParentEnvironmentData{
		Type:         octopusdeployParentEnvironmentsDataType,
		Name:         resourceName,
		ResourceName: resource.Name,
		Ids:          nil,
		Skip:         0,
		Take:         1,
	}
}

// writeData appends the data block for stateless modules
func (c ParentEnvironmentConverter) writeData(file *hclwrite.File, resource octopus.ParentEnvironment, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c ParentEnvironmentConverter) getLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data." + octopusdeployParentEnvironmentsDataType + "." + resourceName + ".parent_environments) != 0 " +
			"? data." + octopusdeployParentEnvironmentsDataType + "." + resourceName + ".parent_environments[0].id " +
			": " + octopusdeployParentEnvironmentResourceType + "." + resourceName + "[0].id}"
	}
	return "${" + octopusdeployParentEnvironmentResourceType + "." + resourceName + ".id}"
}

func (c ParentEnvironmentConverter) getDependency(stateless bool, resourceName string) string {
	if stateless {
		return "${" + octopusdeployParentEnvironmentResourceType + "." + resourceName + "}"
	}

	return ""
}

func (c ParentEnvironmentConverter) getCount(stateless bool, resourceName string) *string {
	if stateless {
		return strutil.StrPointer("${length(data." + octopusdeployParentEnvironmentsDataType + "." + resourceName + ".parent_environments) != 0 ? 0 : 1}")
	}

	return nil
}

// toBashImport creates a bash script to import the resource
func (c ParentEnvironmentConverter) toBashImport(resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --data-urlencode "type=Parent" --header "X-Octopus-ApiKey: $1" "$2/api/$3/environments/v2" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z "${RESOURCE_ID}" ]]
then
	echo "No parent environment found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing parent environment ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, octopusdeployParentEnvironmentResourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c ParentEnvironmentConverter) toPowershellImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/environments/v2?take=10000&type=Parent&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	Write-Error "No parent environment found with the name $ResourceName"
	exit 1
}

echo "Importing parent environment $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, octopusdeployParentEnvironmentResourceType, resourceName), nil
		},
	})
}

func (c ParentEnvironmentConverter) toHcl(environment octopus.ParentEnvironment, _ bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.Excluder.IsResourceExcludedWithRegex(environment.Name, c.ExcludeAllEnvironments, c.ExcludeEnvironments, c.ExcludeEnvironmentsRegex, c.ExcludeEnvironmentsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + environment.Id)
		return nil
	}

	resourceName := "parent_environment_" + sanitizer.SanitizeName(environment.Name)

	if c.GenerateImportScripts && !stateless {
		c.toBashImport(resourceName, environment.Name, dependencies)
		c.toPowershellImport(resourceName, environment.Name, dependencies)
	}

	thisResource := data.ResourceDetails{}
	thisResource.Name = environment.Name
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = environment.Id
	thisResource.SortOrder = environment.SortOrder
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = c.getLookup(stateless, resourceName)
	thisResource.Dependency = c.getDependency(stateless, resourceName)

	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		terraformResource := terraform.TerraformParentEnvironment{
			Type:                        octopusdeployParentEnvironmentResourceType,
			Name:                        resourceName,
			Id:                          strutil.InputPointerIfEnabled(c.IncludeIds, &environment.Id),
			Count:                       c.getCount(stateless, resourceName),
			SpaceId:                     "${trimspace(var.octopus_space_id)}",
			ResourceName:                environment.Name,
			Description:                 strutil.NilIfEmpty(environment.Description),
			UseGuidedFailure:            environment.UseGuidedFailure,
			AutomaticDeprovisioningRule: c.getAutomaticDeprovisioningRule(environment),
		}

		if stateless {
			c.writeData(file, environment, resourceName)
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		// This parent environment must be created after the other parent environments with a lower sort order.
		dependsOn := []string{}
		for _, terraformDependency := range dependencies.GetAllResourceWithLowerSort(c.GetResourceType(), environment.SortOrder) {
			dependency := dependencies.GetResourceDependency(c.GetResourceType(), terraformDependency.Id)
			dependency = hcl.RemoveId(hcl.RemoveInterpolation(dependency))
			dependsOn = append(dependsOn, dependency)
		}
		hcl.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(dependsOn[:], ",")+"]")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c ParentEnvironmentConverter) getAutomaticDeprovisioningRule(environment octopus.ParentEnvironment) *terraform.TerraformAutomaticDeprovisioningRule {
	if environment.AutomaticDeprovisioningRule == nil {
		return nil
	}

	return &terraform.TerraformAutomaticDeprovisioningRule{
		Days:  environment.AutomaticDeprovisioningRule.ExpiryDays,
		Hours: environment.AutomaticDeprovisioningRule.ExpiryHours,
	}

}

func (c ParentEnvironmentConverter) GetResourceType() string {
	return "parentEnvironments"
}
