package converters

import (
	"errors"
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/boolutil"
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
	"k8s.io/utils/strings/slices"
	"regexp"
)

const octopusdeployRunbookResourceType = "octopusdeploy_runbook"

type RunbookConverter struct {
	Client                       client.OctopusClient
	RunbookProcessConverter      ConverterAndLookupByIdAndNameWithProjects
	EnvironmentConverter         ConverterAndLookupWithStatelessById
	ProjectConverter             ConverterAndLookupWithStatelessById
	ExcludedRunbooks             args.StringSliceArgs
	ExcludeRunbooksRegex         args.StringSliceArgs
	ExcludeRunbooksExcept        args.StringSliceArgs
	ExcludeAllRunbooks           bool
	excludeRunbooksRegexCompiled []*regexp.Regexp
	Excluder                     ExcludeByName
	IgnoreProjectChanges         bool
	ErrGroup                     *errgroup.Group
	LimitResourceCount           int
	IncludeSpaceInPopulation     bool
	IncludeIds                   bool
	GenerateImportScripts        bool
}

func (c *RunbookConverter) ToHclByIdWithLookups(id string, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllRunbooks {
		return nil
	}

	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Runbook{}
	foundRunbook, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Runbook: %w", err)
	}

	if !foundRunbook {
		return errors.New("failed to find runbook with id " + id)
	}

	parentResource := octopus.Project{}
	foundProject, err := c.Client.GetSpaceResourceById("Projects", resource.ProjectId, &parentResource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	if !foundProject {
		return errors.New("failed to find project with id " + resource.ProjectId)
	}

	zap.L().Info("Runbook: " + resource.Id + " " + resource.Name)
	return c.toHcl(resource, parentResource.Name, false, true, false, dependencies)
}

func (c *RunbookConverter) ToHclByIdAndName(projectId string, projectName string, recursive bool, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndName(projectId, projectName, recursive, false, dependencies)
}

func (c *RunbookConverter) ToHclStatelessByIdAndName(projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndName(projectId, projectName, true, true, dependencies)
}

func (c *RunbookConverter) toHclByIdAndName(projectId string, projectName string, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllRunbooks {
		return nil
	}

	collection := octopus.GeneralCollection[octopus.Runbook]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetAllResources loading type octopus.GeneralCollection[octopus.Runbook]: %w", err)
	}

	for _, resource := range collection.Items {
		if dependencies.HasResource(resource.Id, c.GetResourceType()) {
			return nil
		}

		zap.L().Info("Runbook: " + resource.Id + " " + resource.Name)
		err = c.toHcl(resource, projectName, recursive, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *RunbookConverter) ToHclLookupByIdAndName(projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllRunbooks {
		return nil
	}

	collection := octopus.GeneralCollection[octopus.Runbook]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetAllResources loading type octopus.GeneralCollection[octopus.Runbook]: %w", err)
	}

	for _, resource := range collection.Items {
		zap.L().Info("Runbook: " + resource.Id + " " + resource.Name)
		err = c.toHcl(resource, projectName, false, true, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

// We consider runbooks to be the responsibility of a project. If the project exists, we don't create the runbook.
func (c *RunbookConverter) buildData(resourceName string, name string) terraform.TerraformProjectData {
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
func (c *RunbookConverter) writeData(file *hclwrite.File, name string, resourceName string) {
	terraformResource := c.buildData(resourceName, name)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

// toBashImport creates a bash script to import the resource
func (c *RunbookConverter) toBashImport(resourceName string, octopusProjectName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Runbooks" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") and select(.ProjectId == \"${PROJECT_ID}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No runbook found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing runbook ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusProjectName, octopusResourceName, octopusdeployRunbookResourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c *RunbookConverter) toPowershellImport(resourceName string, octopusProjectName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Runbooks?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName -and $_.ProjectId -eq $ProjectId} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No runbook found with the name $ResourceName"
	exit 1
}

echo "Importing runbook $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, octopusProjectName, octopusResourceName, octopusdeployRunbookResourceType, resourceName), nil
		},
	})
}

func (c *RunbookConverter) toHcl(runbook octopus.Runbook, projectName string, recursive bool, lookups bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	c.compileRegexes()

	// Ignore excluded runbooks
	if c.Excluder.IsResourceExcludedWithRegex(runbook.Name, c.ExcludeAllRunbooks, c.ExcludedRunbooks, c.ExcludeRunbooksRegex, c.ExcludeRunbooksExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + runbook.Id)
		return nil
	}

	if c.runbookIsExcluded(runbook) {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceNameSuffix := sanitizer.SanitizeName(projectName) + "_" + sanitizer.SanitizeName(runbook.Name)
	runbookName := "runbook_" + resourceNameSuffix

	if c.GenerateImportScripts {
		c.toBashImport(runbookName, projectName, runbook.Name, dependencies)
		c.toPowershellImport(runbookName, projectName, runbook.Name, dependencies)
	}

	err := c.exportChildDependencies(recursive, lookups, stateless, runbook, resourceNameSuffix, dependencies)

	if err != nil {
		return err
	}

	thisResource.FileName = "space_population/" + runbookName + ".tf"
	thisResource.Id = runbook.Id
	thisResource.Name = runbook.Name
	thisResource.ResourceType = c.GetResourceType()

	if stateless {
		// There is no way to look up an existing runbook. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the runbook anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + runbookName + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployRunbookResourceType + "." + runbookName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployRunbookResourceType + "." + runbookName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployRunbookResourceType + "." + runbookName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformRunbook{
			Type:                     octopusdeployRunbookResourceType,
			Name:                     runbookName,
			Id:                       strutil.InputPointerIfEnabled(c.IncludeIds, &runbook.Id),
			SpaceId:                  strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", runbook.SpaceId)),
			ResourceName:             "${var." + runbookName + "_name}",
			ProjectId:                dependencies.GetResource("Projects", runbook.ProjectId),
			EnvironmentScope:         runbook.EnvironmentScope,
			Environments:             dependencies.GetResources("Environments", runbook.Environments...),
			ForcePackageDownload:     runbook.ForcePackageDownload,
			DefaultGuidedFailureMode: runbook.DefaultGuidedFailureMode,
			Description:              strutil.TrimPointer(runbook.Description),
			MultiTenancyMode:         runbook.MultiTenancyMode,
			RetentionPolicy:          c.convertRetentionPolicy(runbook),
			ConnectivityPolicy:       c.convertConnectivityPolicy(runbook),
		}

		file := hclwrite.NewEmptyFile()

		if stateless {
			// when importing a stateless project, the channel is only created if the project does not exist
			c.writeData(file, projectName, runbookName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployProjectsDataType + "." + runbookName + ".projects) != 0 ? 0 : 1}")
		}

		c.writeProjectNameVariable(file, runbookName, runbook.Name)

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)

	return nil
}

func (c *RunbookConverter) GetResourceType() string {
	return "Runbooks"
}

func (c *RunbookConverter) GetGroupResourceType(projectId string) string {
	return "Projects/" + projectId + "/runbooks"
}

func (c *RunbookConverter) writeProjectNameVariable(file *hclwrite.File, projectName string, projectResourceName string) {
	runbookNameVariableResource := terraform.TerraformVariable{
		Name:        projectName + "_name",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The name of the runbook exported from " + projectResourceName,
		Default:     &projectResourceName,
	}

	block := gohcl.EncodeAsBlock(runbookNameVariableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)
}

func (c *RunbookConverter) convertConnectivityPolicy(runbook octopus.Runbook) *terraform.TerraformConnectivityPolicy {
	return &terraform.TerraformConnectivityPolicy{
		AllowDeploymentsToNoTargets: runbook.ConnectivityPolicy.AllowDeploymentsToNoTargets,
		ExcludeUnhealthyTargets:     runbook.ConnectivityPolicy.ExcludeUnhealthyTargets,
		SkipMachineBehavior:         runbook.ConnectivityPolicy.SkipMachineBehavior,
	}
}

func (c *RunbookConverter) convertRetentionPolicy(runbook octopus.Runbook) *terraform.RetentionPolicy {
	return &terraform.RetentionPolicy{
		QuantityToKeep:    runbook.RunRetentionPolicy.QuantityToKeep,
		ShouldKeepForever: boolutil.NilIfFalse(runbook.RunRetentionPolicy.ShouldKeepForever),
	}
}

func (c *RunbookConverter) exportChildDependencies(recursive bool, lookup bool, stateless bool, runbook octopus.Runbook, runbookName string, dependencies *data.ResourceDetailsCollection) error {
	// It is not valid to have lookup be false and recursive be true, as the only supported export of a runbook is
	// with lookup being true.
	if lookup && recursive {
		return errors.New("exporting a runbook with dependencies is not supported")
	}

	// When lookup is true and recursive is false this runbook has been exported as a standalone resource
	// that references its parent project by a lookup.
	// If lookup is true and recursive is true, this runbook was exported with a project, and the project has already
	// been resolved.
	if lookup && !recursive && c.ProjectConverter != nil {
		err := c.ProjectConverter.ToHclLookupById(runbook.ProjectId, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the deployment process
	if runbook.RunbookProcessId != nil {
		var err error
		if lookup {
			err = c.RunbookProcessConverter.ToHclLookupById(*runbook.RunbookProcessId, dependencies)
		} else {
			if stateless {
				err = c.RunbookProcessConverter.ToHclStatelessById(*runbook.RunbookProcessId, dependencies)
			} else {
				err = c.RunbookProcessConverter.ToHclById(*runbook.RunbookProcessId, dependencies)
			}
		}

		if err != nil {
			return err
		}
	}

	for _, e := range runbook.Environments {
		var err error
		if recursive {
			if stateless {
				err = c.EnvironmentConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.EnvironmentConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.EnvironmentConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *RunbookConverter) compileRegexes() {
	if c.ExcludeRunbooksRegex != nil {
		c.excludeRunbooksRegexCompiled = lo.FilterMap(c.ExcludeRunbooksRegex, func(x string, index int) (*regexp.Regexp, bool) {
			re, err := regexp.Compile(x)
			if err != nil {
				return nil, false
			}
			return re, true
		})
	}
}

func (c *RunbookConverter) runbookIsExcluded(runbook octopus.Runbook) bool {
	if c.ExcludedRunbooks != nil && slices.Index(c.ExcludedRunbooks, runbook.Name) != -1 {
		return true
	}

	if c.excludeRunbooksRegexCompiled != nil {
		return lo.SomeBy(c.excludeRunbooksRegexCompiled, func(x *regexp.Regexp) bool {
			return x.MatchString(runbook.Name)
		})
	}

	return false
}
