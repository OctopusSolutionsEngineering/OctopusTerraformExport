package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/boolutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dateutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/intutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"k8s.io/utils/strings/slices"
	"strings"
)

const octopusdeployProjectDeploymentTargetTriggerResourceType = "octopusdeploy_project_deployment_target_trigger"
const octopusdeployProjectScheduledTrigger = "octopusdeploy_project_scheduled_trigger"
const octopusdeployProjectFeedTrigger = "octopusdeploy_external_feed_create_release_trigger"
const octopusdeployProjectGitTrigger = "octopusdeploy_git_trigger"
const octopusdeployProjectArcTrigger = "octopusdeploy_built_in_trigger"

type ProjectTriggerConverter struct {
	Client                client.OctopusClient
	LimitResourceCount    int
	IncludeIds            bool
	GenerateImportScripts bool
	EnvironmentConverter  ConverterAndLookupWithStatelessById
}

func (c ProjectTriggerConverter) ToHclByProjectIdAndName(projectId string, projectName string, recursive bool, lookup bool, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByProjectIdAndName(projectId, projectName, recursive, lookup, false, dependencies)
}

func (c ProjectTriggerConverter) ToHclStatelessByProjectIdAndName(projectId string, projectName string, recursive bool, lookup bool, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByProjectIdAndName(projectId, projectName, recursive, lookup, true, dependencies)
}

func (c ProjectTriggerConverter) toHclByProjectIdAndName(projectId string, projectName string, recursive bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// The API endpoint /api/Spaces-1/projects/Projects-1/triggers does not return ARC triggers
	// You have to add the triggerActionCategory query param to return ARC triggers
	collection := octopus.GeneralCollection[octopus.ProjectTrigger]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection, []string{"triggerActionCategory", "Deployment"})

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetAllResources loading type octopus.GeneralCollection[octopus.ProjectTrigger]: %w", err)
	}

	// You have to add the triggerActionCategory query param to return the runbook triggers
	runbookTriggerCollection := octopus.GeneralCollection[octopus.ProjectTrigger]{}
	err = c.Client.GetAllResources(c.GetGroupResourceType(projectId), &runbookTriggerCollection, []string{"triggerActionCategory", "Runbook"})

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetAllResources loading type octopus.GeneralCollection[octopus.ProjectTrigger]: %w", err)
	}

	// We want all the triggers
	triggers := []octopus.ProjectTrigger{}
	triggers = append(triggers, collection.Items...)
	triggers = append(triggers, runbookTriggerCollection.Items...)

	for _, resource := range triggers {
		if dependencies.HasResource(resource.Id, c.GetResourceType()) {
			return nil
		}

		zap.L().Info("Project Trigger: " + resource.Id + " " + resource.Name)
		err = c.toHcl(resource, recursive, lookup, stateless, projectId, projectName, dependencies)
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
func (c ProjectTriggerConverter) toBashImport(resourceName string, octopusProjectName string, octopusResourceName string, octopusResourceType string, dependencies *data.ResourceDetailsCollection) {
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

PROJECT_NAME="%s"
PROJECT_ID=$(curl --silent -G --data-urlencode "partialName=${PROJECT_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects" | jq -r ".Items[] | select(.Name == \"${PROJECT_NAME}\") | .Id")

if [[ -z PROJECT_ID ]]
then
	echo "No project found with the name ${PROJECT_NAME}"
	exit 1
fi

RESOURCE_NAME="%s"
DEPLOYMENT_RESOURCE_ID=$(curl --silent -G --data-urlencode "triggerActionCategory=Deployment" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects/${PROJECT_ID}/Triggers" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")
RUNBOOK_RESOURCE_ID=$(curl --silent -G --data-urlencode "triggerActionCategory=Runbook" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects/${PROJECT_ID}/Triggers" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z "${DEPLOYMENT_RESOURCE_ID}" && -z "${RUNBOOK_RESOURCE_ID}" ]]
then
	echo "No trigger found with the name ${RESOURCE_NAME}"
	exit 1
fi

RESOURCE_ID=${DEPLOYMENT_RESOURCE_ID:-$RUNBOOK_RESOURCE_ID}

echo "Importing trigger ${RESOURCE_ID}"

ID="%s.%s"
terraform state list "${ID}" &> /dev/null
if [[ $? -ne 0 ]]
then
	terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" ${ID} ${RESOURCE_ID}
fi`,
					resourceName,
					resourceName,
					resourceName,
					resourceName,
					resourceName,
					octopusProjectName,
					octopusResourceName,
					octopusResourceType,
					resourceName),
				nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c ProjectTriggerConverter) toPowershellImport(resourceName string, octopusProjectName string, octopusResourceName string, octopusResourceType string, dependencies *data.ResourceDetailsCollection) {
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

$headers = @{
    "X-Octopus-ApiKey" = $ApiKey
}

$ProjectName="%s"

$ProjectId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ProjectName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ProjectName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ProjectId)) {
	Write-Error "No project found with the name $ProjectName"
	exit 1
}

$ResourceName="%s"

$DeploymentTriggers = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ProjectId/Triggers?triggerActionCategory=Deployment" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items

$RunbookTriggers = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ProjectId/Triggers?triggerActionCategory=Runbook" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items

$AllTriggers = $DeploymentTriggers + $RunbookTriggers

$ResourceId = $AllTriggers | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	Write-Error "No trigger found with the name $ResourceName"
	exit 1
}

echo "Importing trigger $ResourceId"

$Id="%s.%s"
terraform state list "${ID}" *> $null
if ($LASTEXITCODE -ne 0) {
	terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" $Id $ResourceId
}`,
					resourceName,
					octopusProjectName,
					octopusResourceName,
					octopusResourceType,
					resourceName),
				nil
		},
	})
}

func (c ProjectTriggerConverter) toHcl(projectTrigger octopus.ProjectTrigger, recursive bool, lookup bool, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	// Some triggers are not supported
	supportedTriggers := []string{"GitFilter", "ArcFeedFilter", "MachineFilter", "OnceDailySchedule", "FeedFilter", "CronExpressionSchedule", "DaysPerMonthSchedule", "ContinuousDailySchedule", "FeedFilter"}
	if slices.Index(supportedTriggers, projectTrigger.Filter.FilterType) == -1 {
		zap.L().Error("Found an unsupported trigger type " + projectTrigger.Filter.FilterType)
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + projectTrigger.Id)
		return nil
	}

	if err := c.exportEnvironments(projectTrigger, recursive, lookup, stateless, dependencies); err != nil {
		return err
	}

	c.buildTargetTrigger(projectTrigger, stateless, projectId, projectName, dependencies)
	c.buildScheduledTriggerResources(projectTrigger, stateless, projectId, projectName, dependencies)

	err := c.buildArcTriggerResources(projectTrigger, stateless, projectId, projectName, dependencies)

	if err != nil {
		return err
	}

	err = c.buildGitTriggerResources(projectTrigger, stateless, projectId, projectName, dependencies)

	if err != nil {
		return err
	}

	return c.buildFeedTriggerResources(projectTrigger, stateless, projectId, projectName, dependencies)
}

func (c ProjectTriggerConverter) buildTargetTrigger(projectTrigger octopus.ProjectTrigger, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) {
	if projectTrigger.Filter.FilterType != "MachineFilter" {
		return
	}

	projectTriggerName := "projecttrigger_" + sanitizer.SanitizeName(projectName) + "_" + sanitizer.SanitizeName(projectTrigger.Name)

	if c.GenerateImportScripts && !stateless {
		c.toBashImport(projectTriggerName, projectName, projectTrigger.Name, octopusdeployProjectDeploymentTargetTriggerResourceType, dependencies)
		c.toPowershellImport(projectTriggerName, projectName, projectTrigger.Name, octopusdeployProjectDeploymentTargetTriggerResourceType, dependencies)
	}

	thisResource := data.ResourceDetails{}
	thisResource.Name = projectTrigger.Name
	thisResource.FileName = "space_population/" + projectTriggerName + ".tf"
	thisResource.Id = projectTrigger.Id
	thisResource.ResourceType = c.GetGroupResourceType(projectId)
	thisResource.Lookup = "${" + octopusdeployProjectDeploymentTargetTriggerResourceType + "." + projectTriggerName + ".id}"

	if stateless {
		// There is no way to look up an existing trigger. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the trigger anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + ".project_" + sanitizer.SanitizeName(projectName) + ".projects) != 0 " +
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
			EnvironmentIds:  dependencies.GetResources("Environments", projectTrigger.Filter.EnvironmentIds...),
			EventGroups:     projectTrigger.Filter.EventGroups,
			Roles:           projectTrigger.Filter.Roles,
			ShouldRedeploy:  projectTrigger.Action.ShouldRedeployWhenMachineHasBeenDeployedTo,
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			// when importing a stateless project, the trigger is only created if the project does not exist
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployProjectsDataType + ".project_" + sanitizer.SanitizeName(projectName) + ".projects) != 0 ? 0 : 1}")
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
}

func (c ProjectTriggerConverter) buildScheduledTrigger(projectTrigger octopus.ProjectTrigger, projectTriggerName string, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) {
	thisResource := data.ResourceDetails{}
	thisResource.Name = projectTrigger.Name
	thisResource.FileName = "space_population/" + projectTriggerName + ".tf"
	thisResource.Id = projectTrigger.Id
	thisResource.ResourceType = c.GetGroupResourceType(projectId)
	thisResource.Lookup = "${" + octopusdeployProjectScheduledTrigger + "." + projectTriggerName + ".id}"

	if stateless {
		// There is no way to look up an existing trigger. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the trigger anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + ".project_" + sanitizer.SanitizeName(projectName) + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployProjectScheduledTrigger + "." + projectTriggerName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployProjectScheduledTrigger + "." + projectTriggerName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProjectScheduledTrigger + "." + projectTriggerName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		onceDailySchedule, err := c.buildOnceDailySchedule(projectTrigger)

		if err != nil {
			return "", err
		}

		continuousDailySchedule, err := c.buildTerraformProjectScheduledTriggerContinuousDailySchedule(projectTrigger)

		if err != nil {
			return "", err
		}

		daysPerMonthSchedule, err := c.buildTerraformProjectScheduledTriggerDaysPerMonthSchedule(projectTrigger)

		if err != nil {
			return "", err
		}

		terraformResource := terraform.TerraformProjectScheduledTrigger{
			Type:  octopusdeployProjectScheduledTrigger,
			Name:  projectTriggerName,
			Count: nil,
			// Space ID is mandatory in at least 0.18.3, so this field is not dependent on the option to include space IDs
			SpaceId:                   strutil.StrPointer("${trimspace(var.octopus_space_id)}"),
			Id:                        strutil.InputPointerIfEnabled(c.IncludeIds, &projectTrigger.Id),
			ResourceName:              projectTrigger.Name,
			ProjectId:                 dependencies.GetResource("Projects", projectTrigger.ProjectId),
			Description:               strutil.TrimPointer(projectTrigger.Description),
			Timezone:                  projectTrigger.Filter.Timezone,
			IsDisabled:                projectTrigger.IsDisabled,
			ChannelId:                 strutil.NilIfEmptyPointer(dependencies.GetResourcePointer("Channels", projectTrigger.Action.ChannelId)),
			TenantIds:                 dependencies.GetResources("Tenants", projectTrigger.Action.TenantIds...),
			DeployNewReleaseAction:    c.buildDeployNewReleaseAction(projectTrigger, dependencies),
			OnceDailySchedule:         onceDailySchedule,
			ContinuousDailySchedule:   continuousDailySchedule,
			DaysPerMonthSchedule:      daysPerMonthSchedule,
			CronExpressionSchedule:    c.buildTerraformProjectScheduledTriggerCronExpressionSchedule(projectTrigger),
			RunRunbookAction:          c.buildTerraformProjectScheduledTriggerRunRunbookAction(projectTrigger, dependencies),
			DeployLatestReleaseAction: c.buildDeployLatestReleaseAction(projectTrigger, dependencies),
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			// when importing a stateless project, the trigger is only created if the project does not exist
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployProjectsDataType + ".project_" + sanitizer.SanitizeName(projectName) + ".projects) != 0 ? 0 : 1}")
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
}

func (c ProjectTriggerConverter) buildFeedTriggerResources(projectTrigger octopus.ProjectTrigger, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	if projectTrigger.Filter.FilterType != "FeedFilter" {
		return nil
	}

	projectTriggerName := "projecttrigger_" + sanitizer.SanitizeName(projectName) + "_" + sanitizer.SanitizeName(projectTrigger.Name)

	if c.GenerateImportScripts && !stateless {
		c.toBashImport(projectTriggerName, projectName, projectTrigger.Name, octopusdeployProjectFeedTrigger, dependencies)
		c.toPowershellImport(projectTriggerName, projectName, projectTrigger.Name, octopusdeployProjectFeedTrigger, dependencies)
	}

	return c.buildFeedTrigger(projectTrigger, projectTriggerName, stateless, projectId, projectName, dependencies)
}

func (c ProjectTriggerConverter) buildFeedTrigger(projectTrigger octopus.ProjectTrigger, projectTriggerName string, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	project := octopus.Project{}
	_, err := c.Client.GetSpaceResourceById("Projects", projectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	thisResource := data.ResourceDetails{}
	thisResource.Name = projectTrigger.Name
	thisResource.FileName = "space_population/" + projectTriggerName + ".tf"
	thisResource.Id = projectTrigger.Id
	thisResource.ResourceType = c.GetGroupResourceType(projectId)
	thisResource.Lookup = "${" + octopusdeployProjectFeedTrigger + "." + projectTriggerName + ".id}"

	if stateless {
		// There is no way to look up an existing trigger. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the trigger anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + ".project_" + sanitizer.SanitizeName(projectName) + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployProjectFeedTrigger + "." + projectTriggerName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployProjectFeedTrigger + "." + projectTriggerName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProjectFeedTrigger + "." + projectTriggerName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformProjectFeedTrigger{
			Type:  octopusdeployProjectFeedTrigger,
			Name:  projectTriggerName,
			Count: nil,
			// Space ID is mandatory in at least 0.18.3, so this field is not dependent on the option to include space IDs
			SpaceId:      strutil.StrPointer("${trimspace(var.octopus_space_id)}"),
			ProjectId:    dependencies.GetResource("Projects", projectTrigger.ProjectId),
			Id:           strutil.InputPointerIfEnabled(c.IncludeIds, &projectTrigger.Id),
			ResourceName: projectTrigger.Name,
			IsDisabled:   strutil.NilIfFalse(projectTrigger.IsDisabled),
			ChannelId:    strutil.NilIfEmptyPointer(dependencies.GetResourcePointer("Channels", projectTrigger.Action.ChannelId)),
			Package:      c.buildTriggerPackages(projectTrigger),
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			// when importing a stateless project, the trigger is only created if the project does not exist
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployProjectsDataType + ".project_" + sanitizer.SanitizeName(projectName) + ".projects) != 0 ? 0 : 1}")
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		// This trigger needs the deployment process to be created first to ensure step names exist
		if project.DeploymentProcessId != nil {
			hcl.WriteUnquotedAttribute(
				block,
				"depends_on",
				"["+hcl.RemoveId(hcl.RemoveInterpolation(dependencies.GetResourceDependency(
					"DeploymentProcesses/StepOrder",
					strutil.EmptyIfNil(project.DeploymentProcessId))))+"]")
		}

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c ProjectTriggerConverter) buildArcTriggerResources(projectTrigger octopus.ProjectTrigger, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	if projectTrigger.Filter.FilterType != "ArcFeedFilter" {
		return nil
	}

	projectTriggerName := "projecttrigger_" + sanitizer.SanitizeName(projectName) + "_" + sanitizer.SanitizeName(projectTrigger.Name)

	if c.GenerateImportScripts && !stateless {
		c.toBashImport(projectTriggerName, projectName, projectTrigger.Name, octopusdeployProjectArcTrigger, dependencies)
		c.toPowershellImport(projectTriggerName, projectName, projectTrigger.Name, octopusdeployProjectArcTrigger, dependencies)
	}

	return c.buildArcTrigger(projectTrigger, projectTriggerName, stateless, projectId, projectName, dependencies)
}

// getTriggerPackage resolves the step name and package name from the IDs returned by the API for use with the ARC trigger
func (c ProjectTriggerConverter) getTriggerPackage(projectTrigger octopus.ProjectTrigger) (terraform.TerraformBuiltInTriggerPackage, error) {
	releaseCreationPackage := terraform.TerraformBuiltInTriggerPackage{}

	// There should always be a package, but be defensive here
	if len(projectTrigger.Filter.Packages) != 0 {

		// we need the project associated with the trigger
		project := octopus.Project{}
		err := c.Client.GetResourceById("Projects", projectTrigger.ProjectId, &project)

		if err != nil {
			return releaseCreationPackage, err
		}

		// We then need the deployment process associated with the project
		deploymentProcess := octopus.DeploymentProcess{}

		err = c.Client.GetResourceById("DeploymentProcesses", strutil.EmptyIfNil(project.DeploymentProcessId), &deploymentProcess)

		if err != nil {
			return releaseCreationPackage, err
		}

		actions := lo.FlatMap(deploymentProcess.Steps, func(item octopus.Step, index int) []octopus.Action {
			return lo.Filter(item.Actions, func(item octopus.Action, index int) bool {
				return item.Id == projectTrigger.Filter.Packages[0].DeploymentAction
			})
		})

		if len(actions) != 0 {
			action := actions[0]

			// We need the package referenced by the trigger
			pkg, _, exists := lo.FindIndexOf(action.Packages, func(pkg octopus.Package) bool {
				return strutil.EmptyIfNil(pkg.Id) == projectTrigger.Filter.Packages[0].PackageReference
			})

			if exists {
				releaseCreationPackage.PackageReference = strutil.EmptyIfNil(pkg.Name)
				releaseCreationPackage.DeploymentAction = strutil.EmptyIfNil(action.Name)
			}
		}

	}

	return releaseCreationPackage, nil
}

func (c ProjectTriggerConverter) buildArcTrigger(projectTrigger octopus.ProjectTrigger, projectTriggerName string, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	project := octopus.Project{}
	_, err := c.Client.GetSpaceResourceById("Projects", projectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	thisResource := data.ResourceDetails{}
	thisResource.Name = projectTrigger.Name
	thisResource.FileName = "space_population/" + projectTriggerName + ".tf"
	thisResource.Id = projectTrigger.Id
	thisResource.ResourceType = c.GetGroupResourceType(projectId)
	thisResource.Lookup = "${" + octopusdeployProjectArcTrigger + "." + projectTriggerName + ".id}"

	if stateless {
		// There is no way to look up an existing trigger. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the trigger anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + ".project_" + sanitizer.SanitizeName(projectName) + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployProjectArcTrigger + "." + projectTriggerName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployProjectArcTrigger + "." + projectTriggerName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProjectArcTrigger + "." + projectTriggerName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		releaseCreationPackage, err := c.getTriggerPackage(projectTrigger)

		if err != nil {
			return "", err
		}

		terraformResource := terraform.TerraformBuiltInTrigger{
			Type:  octopusdeployProjectArcTrigger,
			Name:  projectTriggerName,
			Count: nil,
			Id:    strutil.InputPointerIfEnabled(c.IncludeIds, &projectTrigger.Id),
			// Space ID is mandatory in at least 0.18.3, so this field is not dependent on the option to include space IDs
			SpaceId:   strutil.StrPointer("${trimspace(var.octopus_space_id)}"),
			ChannelId: dependencies.GetResource("Channels", strutil.EmptyIfNil(projectTrigger.Action.ChannelId)),
			ProjectId: dependencies.GetResource("Projects", projectTrigger.ProjectId),
			// This is defined on the Terraform resource, but doesn't appear in the API
			ReleaseCreationPackageStepId: nil,
			ReleaseCreationPackage:       releaseCreationPackage,
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			// when importing a stateless project, the trigger is only created if the project does not exist
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployProjectArcTrigger + "." + projectName + ".projects) != 0 ? 0 : 1}")
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		// This trigger needs the deployment process to be created first to ensure step names exist
		if project.DeploymentProcessId != nil {
			hcl.WriteUnquotedAttribute(
				block,
				"depends_on",
				"["+hcl.RemoveId(hcl.RemoveInterpolation(dependencies.GetResourceDependency(
					"DeploymentProcesses/StepOrder",
					strutil.EmptyIfNil(project.DeploymentProcessId))))+"]")
		}

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c ProjectTriggerConverter) buildTriggerPackages(projectTrigger octopus.ProjectTrigger) []terraform.TerraformProjectFeedTriggerPackage {
	return lo.Map(projectTrigger.Filter.Packages, func(packageReference octopus.ProjectTriggerFilterPackage, index int) terraform.TerraformProjectFeedTriggerPackage {
		return terraform.TerraformProjectFeedTriggerPackage{
			DeploymentActionSlug: packageReference.DeploymentActionSlug,
			PackageReference:     packageReference.PackageReference,
		}
	})
}

func (c ProjectTriggerConverter) buildScheduledTriggerResources(projectTrigger octopus.ProjectTrigger, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) {
	supportedTypes := []string{"OnceDailySchedule", "CronExpressionSchedule", "DaysPerMonthSchedule", "ContinuousDailySchedule"}

	if slices.Index(supportedTypes, projectTrigger.Filter.FilterType) == -1 {
		return
	}

	projectTriggerName := "projecttrigger_" + sanitizer.SanitizeName(projectName) + "_" + sanitizer.SanitizeName(projectTrigger.Name)

	if c.GenerateImportScripts && !stateless {
		c.toBashImport(projectTriggerName, projectName, projectTrigger.Name, octopusdeployProjectScheduledTrigger, dependencies)
		c.toPowershellImport(projectTriggerName, projectName, projectTrigger.Name, octopusdeployProjectScheduledTrigger, dependencies)
	}

	c.buildScheduledTrigger(projectTrigger, projectTriggerName, stateless, projectId, projectName, dependencies)
}

func (c ProjectTriggerConverter) buildTerraformProjectScheduledTriggerContinuousDailySchedule(projectTrigger octopus.ProjectTrigger) (*terraform.TerraformProjectScheduledTriggerContinuousDailySchedule, error) {
	if projectTrigger.Filter.FilterType != "ContinuousDailySchedule" {
		return nil, nil
	}

	runAfter, err := dateutil.RemoveTimezone(projectTrigger.Filter.RunAfter)

	if err != nil {
		return nil, err
	}

	runUntil, err := dateutil.RemoveTimezone(projectTrigger.Filter.RunUntil)

	if err != nil {
		return nil, err
	}

	return &terraform.TerraformProjectScheduledTriggerContinuousDailySchedule{
		Interval:       strutil.EmptyIfNil(projectTrigger.Filter.Interval),
		RunAfter:       strutil.EmptyIfNil(runAfter),
		RunUntil:       strutil.EmptyIfNil(runUntil),
		HourInterval:   intutil.ZeroIfNil(projectTrigger.Filter.HourInterval),
		MinuteInterval: intutil.ZeroIfNil(projectTrigger.Filter.MinuteInterval),
		DaysOfWeek:     projectTrigger.Filter.DaysOfWeek,
	}, nil
}

func (c ProjectTriggerConverter) buildTerraformProjectScheduledTriggerDaysPerMonthSchedule(projectTrigger octopus.ProjectTrigger) (*terraform.TerraformProjectScheduledTriggerDaysPerMonthSchedule, error) {
	if projectTrigger.Filter.FilterType != "DaysPerMonthSchedule" {
		return nil, nil
	}

	startTime, err := dateutil.RemoveTimezone(projectTrigger.Filter.StartTime)

	if err != nil {
		return nil, err
	}

	return &terraform.TerraformProjectScheduledTriggerDaysPerMonthSchedule{
		MonthlyScheduleType: strutil.EmptyIfNil(projectTrigger.Filter.MonthlyScheduleType),
		StartTime:           strutil.EmptyIfNil(startTime),
		DateOfMonth:         projectTrigger.Filter.DateOfMonth,
		DayNumberOfMonth:    projectTrigger.Filter.DayNumberOfMonth,
		DayOfWeek:           projectTrigger.Filter.DayOfWeek,
	}, nil
}

func (c ProjectTriggerConverter) buildTerraformProjectScheduledTriggerCronExpressionSchedule(projectTrigger octopus.ProjectTrigger) *terraform.TerraformProjectScheduledTriggerCronExpressionSchedule {
	if projectTrigger.Filter.FilterType != "CronExpressionSchedule" {
		return nil
	}

	return &terraform.TerraformProjectScheduledTriggerCronExpressionSchedule{
		CronExpression: strutil.EmptyIfNil(projectTrigger.Filter.CronExpression),
	}
}

func (c ProjectTriggerConverter) buildTerraformProjectScheduledTriggerRunRunbookAction(projectTrigger octopus.ProjectTrigger, dependencies *data.ResourceDetailsCollection) *terraform.TerraformProjectScheduledTriggerRunRunbookAction {
	if projectTrigger.Action.ActionType != "RunRunbook" {
		return nil
	}

	return &terraform.TerraformProjectScheduledTriggerRunRunbookAction{
		TargetEnvironmentIds: dependencies.GetResources("Environments", projectTrigger.Action.EnvironmentIds...),
		RunbookId:            dependencies.GetResource("Runbooks", strutil.EmptyIfNil(projectTrigger.Action.RunbookId)),
	}
}

func (c ProjectTriggerConverter) buildOnceDailySchedule(projectTrigger octopus.ProjectTrigger) (*terraform.TerraformProjectScheduledTriggerDaily, error) {
	if projectTrigger.Filter.FilterType != "OnceDailySchedule" {
		return nil, nil
	}

	dateTime, err := dateutil.RemoveTimezone(projectTrigger.Filter.StartTime)

	if err != nil {
		return nil, err
	}

	return &terraform.TerraformProjectScheduledTriggerDaily{
		StartTime:  strutil.EmptyIfNil(dateTime),
		DaysOfWeek: projectTrigger.Filter.DaysOfWeek,
	}, nil
}

func (c ProjectTriggerConverter) buildDeployNewReleaseAction(projectTrigger octopus.ProjectTrigger, dependencies *data.ResourceDetailsCollection) *terraform.TerraformProjectScheduledTriggerDeployNewReleaseAction {
	if projectTrigger.Action.ActionType != "DeployNewRelease" {
		return nil
	}

	environment := dependencies.GetResource("Environments", strutil.EmptyIfNil(projectTrigger.Action.EnvironmentId))

	return &terraform.TerraformProjectScheduledTriggerDeployNewReleaseAction{
		DestinationEnvironmentId: environment,
	}
}

func (c ProjectTriggerConverter) buildDeployLatestReleaseAction(projectTrigger octopus.ProjectTrigger, dependencies *data.ResourceDetailsCollection) *terraform.TerraformProjectScheduledTriggerDeployLatestReleaseAction {
	if projectTrigger.Action.ActionType != "DeployLatestRelease" {
		return nil
	}

	environment := dependencies.GetResource("Environments", strutil.EmptyIfNil(projectTrigger.Action.DestinationEnvironmentId))
	sourceEnvironments := dependencies.GetResources("Environments", projectTrigger.Action.SourceEnvironmentIds...)

	if len(sourceEnvironments) == 0 {
		zap.L().Info("The source environment was not resolved for trigger " + projectTrigger.Name + ". The resulting trigger has no source environment ID and will not deploy correctly.")
		return nil
	}

	return &terraform.TerraformProjectScheduledTriggerDeployLatestReleaseAction{
		SourceEnvironmentId:      sourceEnvironments[0],
		DestinationEnvironmentId: environment,
		ShouldRedeploy:           boolutil.FalseIfNil(projectTrigger.Action.ShouldRedeployWhenReleaseIsCurrent),
	}
}

func (c ProjectTriggerConverter) GetGroupResourceType(projectId string) string {
	return "Projects/" + projectId + "/Triggers"
}

func (c ProjectTriggerConverter) GetResourceType() string {
	return "ProjectTriggers"
}

func (c ProjectTriggerConverter) exportEnvironments(projectTrigger octopus.ProjectTrigger, recursive bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	environments := []string{}
	environments = append(environments, projectTrigger.Filter.EnvironmentIds...)
	environments = append(environments, projectTrigger.Action.SourceEnvironmentIds...)
	environments = append(environments, projectTrigger.Action.EnvironmentIds...)
	environments = append(environments, strutil.EmptyIfNil(projectTrigger.Action.DestinationEnvironmentId))
	environments = append(environments, strutil.EmptyIfNil(projectTrigger.Action.EnvironmentId))
	environments = lo.Filter(environments, func(environment string, index int) bool {
		return strings.TrimSpace(environment) != ""
	})

	for _, env := range environments {
		if recursive {
			if stateless {
				if err := c.EnvironmentConverter.ToHclStatelessById(env, dependencies); err != nil {
					return err
				}
			} else {
				if err := c.EnvironmentConverter.ToHclById(env, dependencies); err != nil {
					return err
				}
			}
		} else if lookup {
			if err := c.EnvironmentConverter.ToHclLookupById(env, dependencies); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c ProjectTriggerConverter) buildGitTriggerResources(projectTrigger octopus.ProjectTrigger, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	if projectTrigger.Filter.FilterType != "GitFilter" {
		return nil
	}

	projectTriggerName := "projecttrigger_" + sanitizer.SanitizeName(projectName) + "_" + sanitizer.SanitizeName(projectTrigger.Name)

	if c.GenerateImportScripts && !stateless {
		c.toBashImport(projectTriggerName, projectName, projectTrigger.Name, octopusdeployProjectGitTrigger, dependencies)
		c.toPowershellImport(projectTriggerName, projectName, projectTrigger.Name, octopusdeployProjectGitTrigger, dependencies)
	}

	return c.buildGitTrigger(projectTrigger, projectTriggerName, stateless, projectId, projectName, dependencies)
}

func (c ProjectTriggerConverter) buildGitTrigger(projectTrigger octopus.ProjectTrigger, projectTriggerName string, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	project := octopus.Project{}
	_, err := c.Client.GetSpaceResourceById("Projects", projectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	thisResource := data.ResourceDetails{}
	thisResource.Name = projectTrigger.Name
	thisResource.FileName = "space_population/" + projectTriggerName + ".tf"
	thisResource.Id = projectTrigger.Id
	thisResource.ResourceType = c.GetGroupResourceType(projectId)
	thisResource.Lookup = "${" + octopusdeployProjectGitTrigger + "." + projectTriggerName + ".id}"

	if stateless {
		// There is no way to look up an existing trigger. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the trigger anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + ".project_" + sanitizer.SanitizeName(projectName) + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployProjectGitTrigger + "." + projectTriggerName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployProjectGitTrigger + "." + projectTriggerName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProjectGitTrigger + "." + projectTriggerName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformGitTrigger{
			Type:  octopusdeployProjectGitTrigger,
			Name:  projectTriggerName,
			Id:    strutil.InputPointerIfEnabled(c.IncludeIds, &projectTrigger.Id),
			Count: nil,
			// Space ID is mandatory in at least 0.18.3, so this field is not dependent on the option to include space IDs
			SpaceId:      strutil.StrPointer("${trimspace(var.octopus_space_id)}"),
			ResourceName: projectTrigger.Name,
			Description:  projectTrigger.Description,
			ProjectId:    dependencies.GetResource("Projects", projectTrigger.ProjectId),
			ChannelId:    strutil.EmptyIfNil(dependencies.GetResourcePointer("Channels", projectTrigger.Action.ChannelId)),
			IsDisabled:   strutil.NilIfFalse(projectTrigger.IsDisabled),
			Sources: lo.Map(projectTrigger.Sources, func(item octopus.ProjectTriggerSources, index int) terraform.TerraformGitTriggerSource {
				return terraform.TerraformGitTriggerSource{
					DeploymentActionSlug: item.DeploymentActionSlug,
					ExcludeFilePaths:     item.ExcludeFilePaths,
					GitDependencyName:    item.GitDependencyName,
					IncludeFilePaths:     item.IncludeFilePaths,
				}
			}),
		}

		file := hclwrite.NewEmptyFile()

		if stateless {
			// when importing a stateless project, the trigger is only created if the project does not exist
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployProjectsDataType + ".project_" + sanitizer.SanitizeName(projectName) + ".projects) != 0 ? 0 : 1}")
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		// This trigger needs the deployment process to be created first to ensure step names exist
		if project.DeploymentProcessId != nil {
			hcl.WriteUnquotedAttribute(
				block,
				"depends_on",
				"["+hcl.RemoveId(hcl.RemoveInterpolation(dependencies.GetResourceDependency(
					"DeploymentProcesses/StepOrder",
					strutil.EmptyIfNil(project.DeploymentProcessId))))+"]")
		}

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}
