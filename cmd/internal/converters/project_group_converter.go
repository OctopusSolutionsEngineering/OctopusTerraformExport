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

const octopusdeployProjectGroupsDataType = "octopusdeploy_project_groups"
const octopusdeployProjectGroupResourceType = "octopusdeploy_project_group"
const defaultProjectGroup = "Default Project Group"

type ProjectGroupConverter struct {
	Client                     client.OctopusClient
	ErrGroup                   *errgroup.Group
	ExcludeProjectGroups       args.StringSliceArgs
	ExcludeProjectGroupsRegex  args.StringSliceArgs
	ExcludeProjectGroupsExcept args.StringSliceArgs
	ExcludeAllProjectGroups    bool
	Excluder                   ExcludeByName
	LimitResourceCount         int
	IncludeSpaceInPopulation   bool
	IncludeIds                 bool
	GenerateImportScripts      bool
}

func (c ProjectGroupConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c ProjectGroupConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c ProjectGroupConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllProjectGroups {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.ProjectGroup]{
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

		if c.Excluder.IsResourceExcludedWithRegex(
			resource.Name,
			c.ExcludeAllProjectGroups,
			c.ExcludeProjectGroups,
			c.ExcludeProjectGroupsRegex,
			c.ExcludeProjectGroupsExcept) {
			continue
		}

		zap.L().Info("Project Group: " + resource.Id + " " + resource.Name)
		err := c.toHcl(resource, false, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c ProjectGroupConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c ProjectGroupConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c ProjectGroupConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {

	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.ProjectGroup{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.ProjectGroup: %w", err)
	}

	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllProjectGroups, c.ExcludeProjectGroups, c.ExcludeProjectGroupsRegex, c.ExcludeProjectGroupsExcept) {
		return nil
	}

	zap.L().Info("Project Group: " + resource.Id + " " + resource.Name)
	return c.toHcl(resource, false, false, stateless, dependencies)
}

func (c ProjectGroupConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.ProjectGroup{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.ProjectGroup: %w", err)
	}

	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllProjectGroups, c.ExcludeProjectGroups, c.ExcludeProjectGroupsRegex, c.ExcludeProjectGroupsExcept) {
		return nil
	}

	return c.toHcl(resource, false, true, false, dependencies)
}

func (c ProjectGroupConverter) buildData(resourceName string, name string) terraform.TerraformProjectGroupData {
	return terraform.TerraformProjectGroupData{
		Type:        octopusdeployProjectGroupsDataType,
		Name:        name,
		Ids:         nil,
		PartialName: resourceName,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c ProjectGroupConverter) writeData(file *hclwrite.File, name string, resourceName string) {
	terraformResource := c.buildData(resourceName, name)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

// toBashImport creates a bash script to import the resource
func (c ProjectGroupConverter) toBashImport(resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/ProjectGroups" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No project group found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing project group ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, octopusdeployProjectGroupResourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c ProjectGroupConverter) toPowershellImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/ProjectGroups?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No project group found with the name $ResourceName"
	exit 1
}

echo "Importing project group $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, octopusdeployProjectGroupResourceType, resourceName), nil
		},
	})
}

func (c ProjectGroupConverter) toHcl(resource octopus.ProjectGroup, recursive bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllProjectGroups, c.ExcludeProjectGroups, c.ExcludeProjectGroupsRegex, c.ExcludeProjectGroupsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + resource.Id)
		return nil
	}

	thisResource := data.ResourceDetails{}

	forceLookup := lookup || resource.Name == defaultProjectGroup

	projectName := "project_group_" + sanitizer.SanitizeName(resource.Name)

	if c.GenerateImportScripts && resource.Name != defaultProjectGroup && !lookup && !stateless {
		c.toBashImport(projectName, resource.Name, dependencies)
		c.toPowershellImport(projectName, resource.Name, dependencies)
	}

	thisResource.FileName = "space_population/projectgroup_" + projectName + ".tf"
	thisResource.Id = resource.Id
	thisResource.Name = resource.Name
	thisResource.ResourceType = c.GetResourceType()

	if forceLookup {
		thisResource.Lookup = "${data." + octopusdeployProjectGroupsDataType + "." + projectName + ".project_groups[0].id}"
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData("${var."+projectName+"_name}", projectName)
			file := hclwrite.NewEmptyFile()
			c.writeProjectNameVariable(file, projectName, resource.Name)
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a project group called ${var."+projectName+"_name}. This resource must exist in the space before this Terraform configuration is applied.", "length(self.project_groups) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	} else {
		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployProjectGroupsDataType + "." + projectName + ".project_groups) != 0 " +
				"? data." + octopusdeployProjectGroupsDataType + "." + projectName + ".project_groups[0].id " +
				": " + octopusdeployProjectGroupResourceType + "." + projectName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployProjectGroupResourceType + "." + projectName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeployProjectGroupResourceType + "." + projectName + ".id}"
		}

		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform.TerraformProjectGroup{
				Id:           strutil.InputPointerIfEnabled(c.IncludeIds, &resource.Id),
				SpaceId:      strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", resource.SpaceId)),
				Type:         octopusdeployProjectGroupResourceType,
				Name:         projectName,
				ResourceName: "${var." + projectName + "_name}",
				Description:  strutil.TrimPointer(resource.Description),
			}
			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, projectName, "${var."+projectName+"_name}")
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployProjectGroupsDataType + "." + projectName + ".project_groups) != 0 ? 0 : 1}")
			}

			c.writeProjectNameVariable(file, projectName, resource.Name)

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			if stateless {
				hcl.WriteLifecyclePreventDestroyAttribute(block)
			}

			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	}

	if recursive {
		// export child projects
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c ProjectGroupConverter) writeProjectNameVariable(file *hclwrite.File, projectName string, projectGroupResourceName string) {
	projectNameVariableResource := terraform.TerraformVariable{
		Name:        projectName + "_name",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The name of the project group to lookup",
		Default:     &projectGroupResourceName,
	}

	block := gohcl.EncodeAsBlock(projectNameVariableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)
}

func (c ProjectGroupConverter) GetResourceType() string {
	return "ProjectGroups"
}
