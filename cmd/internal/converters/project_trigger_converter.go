package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

const octopusdeployProjectDeploymentTargetTriggerResourceType = "octopusdeploy_project_deployment_target_trigger"

type ProjectTriggerConverter struct {
	Client             client.OctopusClient
	LimitResourceCount int
	IncludeIds         bool
}

func (c ProjectTriggerConverter) ToHclByProjectIdAndName(projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.ProjectTrigger]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Project Trigger: " + resource.Id)
		err = c.toHcl(resource, false, false, projectId, projectName, dependencies)
		if err != nil {
			return err
		}
	}

	return nil
}

// We consider triggers to be the responsibility of a project. If the project exists, we don't create the trigger.
func (c ProjectTriggerConverter) buildData(resourceName string, name string) terraform.TerraformProjectData {
	return terraform.TerraformProjectData{
		Type:        octopusdeployProjectsDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c ProjectTriggerConverter) writeData(file *hclwrite.File, name string, resourceName string) {
	terraformResource := c.buildData(resourceName, name)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

// toBashImport creates a bash script to import the resource
func (c ProjectTriggerConverter) toBashImport(resourceName string, octopusProjectName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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

PROJECT_NAME="%s"
PROJECT_ID=$(curl --silent -G --data-urlencode "partialName=${PROJECT_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects" | jq -r ".Items[] | select(.Name == \"${PROJECT_NAME}\") | .Id")

if [[ -z PROJECT_ID ]]
then
	echo "No project found with the name ${PROJECT_NAME}"
	exit 1
fi

RESOURCE_NAME="%s"
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects/${PROJECT_ID}/Triggers" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No trigger found with the name ${RESOURCE_ID}"
	exit 1
fi

echo "Importing trigger ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusProjectName, octopusResourceName, octopusdeployProjectDeploymentTargetTriggerResourceType, resourceName), nil
		},
	})
}

func (c ProjectTriggerConverter) toHcl(projectTrigger octopus2.ProjectTrigger, _ bool, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	// Scheduled triggers with types like "OnceDailySchedule" are not supported
	if projectTrigger.Filter.FilterType != "MachineFilter" {
		zap.L().Error("Found an unsupported trigger type " + projectTrigger.Filter.FilterType)
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + projectTrigger.Id)
		return nil
	}

	projectTriggerName := "projecttrigger_" + sanitizer.SanitizeName(projectName) + "_" + sanitizer.SanitizeName(projectTrigger.Name)

	c.toBashImport(projectTriggerName, projectName, projectTrigger.Name, dependencies)

	thisResource := data.ResourceDetails{}
	thisResource.Name = projectTrigger.Name
	thisResource.FileName = "space_population/" + projectTriggerName + ".tf"
	thisResource.Id = projectTrigger.Id
	thisResource.ResourceType = c.GetGroupResourceType(projectId)
	thisResource.Lookup = "${" + octopusdeployProjectDeploymentTargetTriggerResourceType + "." + projectTriggerName + ".id}"

	if stateless {
		// There is no way to look up an existing trigger. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the trigger anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + projectTriggerName + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployProjectDeploymentTargetTriggerResourceType + "." + projectTriggerName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployProjectDeploymentTargetTriggerResourceType + "." + projectTriggerName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProjectDeploymentTargetTriggerResourceType + "." + projectTriggerName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformProjectTrigger{
			Type:            octopusdeployProjectDeploymentTargetTriggerResourceType,
			Name:            projectTriggerName,
			Id:              strutil.InputPointerIfEnabled(c.IncludeIds, &projectTrigger.Id),
			ResourceName:    projectTrigger.Name,
			ProjectId:       dependencies.GetResource("Projects", projectTrigger.ProjectId),
			EventCategories: projectTrigger.Filter.EventCategories,
			EnvironmentIds:  projectTrigger.Filter.EnvironmentIds,
			EventGroups:     projectTrigger.Filter.EventGroups,
			Roles:           projectTrigger.Filter.Roles,
			ShouldRedeploy:  projectTrigger.Action.ShouldRedeployWhenMachineHasBeenDeployedTo,
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			// when importing a stateless project, the trigger is only created if the project does not exist
			c.writeData(file, projectName, projectTriggerName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployProjectsDataType + "." + projectTriggerName + ".projects) != 0 ? 0 : 1}")
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

func (c ProjectTriggerConverter) GetGroupResourceType(projectId string) string {
	return "Projects/" + projectId + "/Triggers"
}

func (c ProjectTriggerConverter) GetResourceType() string {
	return "ProjectTriggers"
}
