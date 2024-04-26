package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/intutil"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"k8s.io/utils/strings/slices"
	"time"
)

const octopusdeployProjectDeploymentTargetTriggerResourceType = "octopusdeploy_project_deployment_target_trigger"
const octopusdeployProjectScheduledTrigger = "octopusdeploy_project_scheduled_trigger"

type ProjectTriggerConverter struct {
	Client                client.OctopusClient
	LimitResourceCount    int
	IncludeIds            bool
	GenerateImportScripts bool
	EnvironmentFilter     EnvironmentFilter
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
	echo "No trigger found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing trigger ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusProjectName, octopusResourceName, octopusResourceType, resourceName), nil
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
	echo "No project found with the name $ProjectName"
	exit 1
}

$ResourceName="%s"

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ProjectId/Triggers?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No target found with the name $ResourceName"
	exit 1
}

echo "Importing target $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, octopusProjectName, octopusResourceName, octopusResourceType, resourceName), nil
		},
	})
}

func (c ProjectTriggerConverter) toHcl(projectTrigger octopus2.ProjectTrigger, _ bool, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	// Scheduled triggers with types like "OnceDailySchedule" are not supported
	supportedTriggers := []string{"MachineFilter", "OnceDailySchedule", "FeedFilter", "CronExpressionSchedule", "DaysPerMonthSchedule"}
	if slices.Index(supportedTriggers, projectTrigger.Filter.FilterType) == -1 {
		zap.L().Error("Found an unsupported trigger type " + projectTrigger.Filter.FilterType)
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + projectTrigger.Id)
		return nil
	}

	c.buildTargetTrigger(projectTrigger, stateless, projectId, projectName, dependencies)
	c.buildScheduledTriggerResources(projectTrigger, stateless, projectId, projectName, dependencies)

	return nil
}

func (c ProjectTriggerConverter) buildTargetTrigger(projectTrigger octopus2.ProjectTrigger, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) {
	if projectTrigger.Filter.FilterType != "MachineFilter" {
		return
	}

	projectTriggerName := "projecttrigger_" + sanitizer.SanitizeName(projectName) + "_" + sanitizer.SanitizeName(projectTrigger.Name)

	if c.GenerateImportScripts {
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
}

func (c ProjectTriggerConverter) buildScheduledTrigger(projectTrigger octopus2.ProjectTrigger, projectTriggerName string, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) {
	thisResource := data.ResourceDetails{}
	thisResource.Name = projectTrigger.Name
	thisResource.FileName = "space_population/" + projectTriggerName + ".tf"
	thisResource.Id = projectTrigger.Id
	thisResource.ResourceType = c.GetGroupResourceType(projectId)
	thisResource.Lookup = "${" + octopusdeployProjectScheduledTrigger + "." + projectTriggerName + ".id}"

	if stateless {
		// There is no way to look up an existing trigger. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the trigger anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + projectTriggerName + ".projects) != 0 " +
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

		runBookAction, include := c.buildTerraformProjectScheduledTriggerRunRunbookAction(projectTrigger, dependencies)

		if !include {
			return "", nil
		}

		deployNewReleaseAction, include := c.buildDeployNewReleaseAction(projectTrigger, dependencies)

		if !include {
			return "", nil
		}

		terraformResource := terraform.TerraformProjectScheduledTrigger{
			Type:  octopusdeployProjectScheduledTrigger,
			Name:  projectTriggerName,
			Count: nil,
			// Space ID is mandatory in at least 0.18.3, so this field is not dependent on the option to include space IDs
			SpaceId:                 strutil.StrPointer("${trimspace(var.octopus_space_id)}"),
			Id:                      strutil.InputPointerIfEnabled(c.IncludeIds, &projectTrigger.Id),
			ResourceName:            projectTrigger.Name,
			ProjectId:               dependencies.GetResource("Projects", projectTrigger.ProjectId),
			DeployNewReleaseAction:  deployNewReleaseAction,
			OnceDailySchedule:       onceDailySchedule,
			ContinuousDailySchedule: c.buildTerraformProjectScheduledTriggerContinuousDailySchedule(projectTrigger),
			CronExpressionSchedule:  c.buildTerraformProjectScheduledTriggerCronExpressionSchedule(projectTrigger),
			RunRunbookAction:        runBookAction,
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
}

func (c ProjectTriggerConverter) buildScheduledTriggerResources(projectTrigger octopus2.ProjectTrigger, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) {
	if projectTrigger.Filter.FilterType != "OnceDailySchedule" && projectTrigger.Filter.FilterType != "CronExpressionSchedule" && projectTrigger.Filter.FilterType != "DaysPerMonthSchedule" {
		return
	}

	projectTriggerName := "projecttrigger_" + sanitizer.SanitizeName(projectName) + "_" + sanitizer.SanitizeName(projectTrigger.Name)

	if c.GenerateImportScripts {
		c.toBashImport(projectTriggerName, projectName, projectTrigger.Name, octopusdeployProjectScheduledTrigger, dependencies)
		c.toPowershellImport(projectTriggerName, projectName, projectTrigger.Name, octopusdeployProjectScheduledTrigger, dependencies)
	}

	c.buildScheduledTrigger(projectTrigger, projectTriggerName, stateless, projectId, projectName, dependencies)
}

func (c ProjectTriggerConverter) buildTerraformProjectScheduledTriggerContinuousDailySchedule(projectTrigger octopus2.ProjectTrigger) *terraform.TerraformProjectScheduledTriggerContinuousDailySchedule {
	if projectTrigger.Filter.StartTime != nil || // A start time indicates a once_daily_schedule block
		(projectTrigger.Filter.Interval == nil &&
			projectTrigger.Filter.HourInterval == nil &&
			projectTrigger.Filter.MinuteInterval == nil &&
			projectTrigger.Filter.RunAfter == nil &&
			projectTrigger.Filter.RunUntil == nil &&
			(projectTrigger.Filter.DaysOfWeek == nil || len(projectTrigger.Filter.DaysOfWeek) == 0)) {
		return nil
	}

	return &terraform.TerraformProjectScheduledTriggerContinuousDailySchedule{
		Interval:       strutil.EmptyIfNil(projectTrigger.Filter.Interval),
		RunAfter:       strutil.EmptyIfNil(projectTrigger.Filter.RunAfter),
		RunUntil:       strutil.EmptyIfNil(projectTrigger.Filter.RunUntil),
		HourInterval:   intutil.ZeroIfNil(projectTrigger.Filter.HourInterval),
		MinuteInterval: intutil.ZeroIfNil(projectTrigger.Filter.MinuteInterval),
		DaysOfWeek:     projectTrigger.Filter.DaysOfWeek,
	}
}

func (c ProjectTriggerConverter) buildTerraformProjectScheduledTriggerCronExpressionSchedule(projectTrigger octopus2.ProjectTrigger) *terraform.TerraformProjectScheduledTriggerCronExpressionSchedule {
	if projectTrigger.Filter.CronExpression == nil {
		return nil
	}

	return &terraform.TerraformProjectScheduledTriggerCronExpressionSchedule{
		CronExpression: strutil.EmptyIfNil(projectTrigger.Filter.CronExpression),
	}
}

func (c ProjectTriggerConverter) buildTerraformProjectScheduledTriggerRunRunbookAction(projectTrigger octopus2.ProjectTrigger, dependencies *data.ResourceDetailsCollection) (*terraform.TerraformProjectScheduledTriggerRunRunbookAction, bool) {
	if strutil.NilIfEmptyPointer(projectTrigger.Action.RunbookId) == nil &&
		(projectTrigger.Action.EnvironmentIds == nil || len(projectTrigger.Action.EnvironmentIds) == 0) {
		return nil, true
	}

	environments := dependencies.GetResources("Environments", c.EnvironmentFilter.FilterEnvironmentScope(projectTrigger.Action.EnvironmentIds)...)
	runbook := dependencies.GetResource("Runbooks", strutil.EmptyIfNil(projectTrigger.Action.RunbookId))

	// This means the resources that the target triggers were filtered out, so we need to exclude this trigger
	if runbook == "" || len(environments) == 0 {
		return nil, false
	}

	return &terraform.TerraformProjectScheduledTriggerRunRunbookAction{
		TargetEnvironmentIds: dependencies.GetResources("Environments", c.EnvironmentFilter.FilterEnvironmentScope(projectTrigger.Action.EnvironmentIds)...),
		RunbookId:            dependencies.GetResource("Runbooks", strutil.EmptyIfNil(projectTrigger.Action.RunbookId)),
	}, true
}

func (c ProjectTriggerConverter) buildOnceDailySchedule(projectTrigger octopus2.ProjectTrigger) (*terraform.TerraformProjectScheduledTriggerDaily, error) {
	if strutil.NilIfEmptyPointer(projectTrigger.Filter.Interval) != nil || // An interval indicates a continuous_daily_schedule block
		(strutil.NilIfEmptyPointer(projectTrigger.Filter.StartTime) == nil &&
			(projectTrigger.Filter.DaysOfWeek == nil || len(projectTrigger.Filter.DaysOfWeek) == 0)) {
		return nil, nil
	}

	// The TF provider doesn't like the time formats returned by the Octopus API.
	// We need to parse the date so we can serialize it in a format accepted by the TF provider.
	dateTime, err := time.Parse("2006-01-02T15:04:05.000Z07:00", strutil.EmptyIfNil(projectTrigger.Filter.StartTime))

	if err != nil {
		return nil, err
	}

	// The TF provider responds with this error when using a ISO RFC3339 date format:
	// Error: parsing time "2024-03-22T09:00:00Z": extra text: "Z"
	// So we need to format the date without the timezone.
	return &terraform.TerraformProjectScheduledTriggerDaily{
		StartTime:  dateTime.Format("2006-01-02T15:04:05"),
		DaysOfWeek: projectTrigger.Filter.DaysOfWeek,
	}, nil
}

func (c ProjectTriggerConverter) buildDeployNewReleaseAction(projectTrigger octopus2.ProjectTrigger, dependencies *data.ResourceDetailsCollection) (*terraform.TerraformProjectScheduledTriggerDeployNewReleaseAction, bool) {
	if strutil.NilIfEmptyPointer(projectTrigger.Action.DestinationEnvironmentId) == nil &&
		strutil.NilIfEmptyPointer(projectTrigger.Action.EnvironmentId) == nil {
		return nil, true
	}

	environment := dependencies.GetResource("Environments", strutil.DefaultIfEmptyOrNil(projectTrigger.Action.DestinationEnvironmentId, strutil.EmptyIfNil(projectTrigger.Action.EnvironmentId)))

	if environment == "" {
		return nil, false
	}

	return &terraform.TerraformProjectScheduledTriggerDeployNewReleaseAction{
		DestinationEnvironmentId: environment,
	}, true
}

func (c ProjectTriggerConverter) GetGroupResourceType(projectId string) string {
	return "Projects/" + projectId + "/Triggers"
}

func (c ProjectTriggerConverter) GetResourceType() string {
	return "ProjectTriggers"
}
