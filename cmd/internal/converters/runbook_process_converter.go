package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"go.uber.org/zap"
	"net/url"
)

// RunbookProcessConverter converts deployment processes for v1 of the Octopus Terraform provider.
type RunbookProcessConverter struct {
	DeploymentProcessConverterBase
}

func (c *RunbookProcessConverter) ToHclByIdBranchAndProject(parentId string, runbookProcessId string, branch string, recursive bool, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdBranchAndProject(parentId, runbookProcessId, branch, recursive, false, dependencies)
}

func (c *RunbookProcessConverter) ToHclStatelessByIdBranchAndProject(parentId string, runbookProcessId string, branch string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdBranchAndProject(parentId, runbookProcessId, branch, true, true, dependencies)
}

func (c *RunbookProcessConverter) toHclByIdBranchAndProject(parentId string, runbookProcessId string, branch string, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if parentId == "" || branch == "" {
		return nil
	}

	if dependencies.HasResource(parentId+"/"+branch, c.GetResourceType()) {
		return nil
	}

	// Get the deployment process associated with the git branch
	resource := octopus.RunbookProcess{}
	found, err := c.Client.GetResource("Projects/"+parentId+"/"+url.QueryEscape(branch)+"/runbookProcesses/"+runbookProcessId, &resource)

	if err != nil {
		if !c.IgnoreCacErrors {
			return err
		} else {
			found = false
		}
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	runbook := octopus.Runbook{}
	found, err = c.Client.GetSpaceResourceById("Projects/"+parentId+"/"+url.QueryEscape(branch)+"/Runbooks", resource.RunbookId, &runbook)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Runbook: %w", err)
	}

	if !found {
		return fmt.Errorf("runbook with ID %s not found in project %s", resource.RunbookId, parentId+"/"+branch)
	}

	project := octopus.Project{}
	found, err = c.Client.GetSpaceResourceById("Projects", parentId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	if !found {
		return fmt.Errorf("project with ID %s not found", parentId)
	}

	c.exportScripts(project, runbook, resource, dependencies)

	return c.toHcl(&resource, &project, &runbook, recursive, false, stateless, dependencies)
}

func (c *RunbookProcessConverter) ToHclLookupByIdBranchAndProject(parentId string, runbookProcessId string, branch string, dependencies *data.ResourceDetailsCollection) error {
	if parentId == "" || branch == "" {
		return nil
	}

	if dependencies.HasResource(parentId+"/"+branch, c.GetResourceType()) {
		return nil
	}

	// Get the deployment process associated with the git branch
	resource := octopus.RunbookProcess{}
	found, err := c.Client.GetResource("Projects/"+parentId+"/"+url.QueryEscape(branch)+"/runbookProcesses/"+runbookProcessId, &resource)

	if err != nil {
		if !c.IgnoreCacErrors {
			return err
		} else {
			found = false
		}
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	runbook := octopus.Runbook{}
	_, err = c.Client.GetSpaceResourceById("Runbooks", resource.RunbookId, &runbook)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Runbook: %w", err)
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", runbook.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	c.exportScripts(project, runbook, resource, dependencies)

	return c.toHcl(&resource, &project, &runbook, false, true, false, dependencies)
}

func (c *RunbookProcessConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, false, dependencies)
}

func (c *RunbookProcessConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, true, dependencies)
}

func (c *RunbookProcessConverter) toHclById(id string, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.RunbookProcess{}
	found, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.RunbookProcess: %w", err)
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	runbook := octopus.Runbook{}
	_, err = c.Client.GetSpaceResourceById("Runbooks", resource.RunbookId, &runbook)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Runbook: %w", err)
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", runbook.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	zap.L().Info("Deployment Process: " + resource.Id)

	c.exportScripts(project, runbook, resource, dependencies)

	return c.toHcl(&resource, &project, &runbook, recursive, false, stateless, dependencies)
}

func (c *RunbookProcessConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.RunbookProcess{}
	found, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.RunbookProcess: %w", err)
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	runbook := octopus.Runbook{}
	_, err = c.Client.GetSpaceResourceById("Runbooks", resource.RunbookId, &runbook)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Runbook: %w", err)
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", runbook.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	c.exportScripts(project, runbook, resource, dependencies)

	return c.toHcl(&resource, &project, &runbook, false, true, false, dependencies)
}

func (c *RunbookProcessConverter) exportScripts(project octopus.Project, runbook octopus.Runbook, resource octopus.RunbookProcess, dependencies *data.ResourceDetailsCollection) {
	if c.GenerateImportScripts {
		c.toBashImport(c.generateProcessName(&project, &runbook), c.generateStepOrderName(&project, &runbook), project.GetName(), runbook.GetName(), dependencies)
		c.toPowershellImport(c.generateProcessName(&project, &runbook), c.generateStepOrderName(&project, &runbook), project.GetName(), runbook.GetName(), dependencies)

		validSteps := c.getValidSteps(&resource)

		for _, step := range validSteps {
			c.toStepBashImport(c.generateStepName(&project, &runbook, &step), project.GetName(), runbook.GetName(), step.GetName(), dependencies)
			c.toStepPowershellImport(c.generateStepName(&project, &runbook, &step), project.GetName(), runbook.GetName(), step.GetName(), dependencies)

			for _, action := range step.Actions[1:] {
				c.toChildStepBashImport(c.generateChildStepName(&project, &runbook, &action), project.GetName(), runbook.GetName(), step.GetName(), action.GetName(), dependencies)
				c.toChildStepPowershellImport(c.generateChildStepName(&project, &runbook, &action), project.GetName(), runbook.GetName(), step.GetName(), action.GetName(), dependencies)
			}
		}
	}
}

// toBashImport creates a bash script to import the resource
func (c *RunbookProcessConverter) toBashImport(resourceName string, stepsOrderResourceName string, octopusProjectName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Runbooks" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\" and .ProjectId == \"${PROJECT_ID}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No runbook found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing runbook ${RESOURCE_ID}"

ID="%s.%s"
terraform state list "${ID}" &> /dev/null
if [[ $? -ne 0 ]]
then
	terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" "${ID}" RunbookProcess-${RESOURCE_ID}
fi

ID="%s.%s"
terraform state list "${ID}" &> /dev/null
if [[ $? -ne 0 ]]
then
	terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" "${ID}" RunbookProcess-${RESOURCE_ID}
fi`,
					resourceName,
					resourceName,
					resourceName,
					resourceName,
					resourceName,
					octopusProjectName,
					octopusResourceName,
					octopusdeployProcessResourceType,
					resourceName,
					octopusdeployProcessStepsOrderResourceType,
					stepsOrderResourceName),
				nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c *RunbookProcessConverter) toPowershellImport(resourceName string, stepsOrderResourceName string, projectName string, runbookName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ProjectId/Runbooks?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No runbook found with the name $ResourceName"
	exit 1
}

echo "Importing runbook $ResourceId"

$Id="%s.%s"
terraform state list "${ID}" *> $null
if ($LASTEXITCODE -ne 0) {
	terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" $Id "RunbookProcess-$ResourceId"
}

$Id="%s.%s"
terraform state list "${ID}" *> $null
if ($LASTEXITCODE -ne 0) {
	terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" $Id "RunbookProcess-$ResourceId"
}`,
					resourceName,
					projectName,
					runbookName,
					octopusdeployProcessResourceType,
					resourceName,
					octopusdeployProcessStepsOrderResourceType,
					stepsOrderResourceName),
				nil
		},
	})
}

// toStepBashImport creates a bash script to import the step resource
func (c *RunbookProcessConverter) toStepBashImport(resourceName string, projectName string, runbookName string, stepName string, dependencies *data.ResourceDetailsCollection) {
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

if [[ -z "${PROJECT_ID}" ]]
then
	echo "No project found with the name ${PROJECT_NAME}"
	exit 1
fi

RESOURCE_NAME="%s"
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects/${PROJECT_ID}/Runbooks" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z "${RESOURCE_ID}" ]]
then
	echo "No project found with the name ${RESOURCE_NAME}"
	exit 1
fi

# The step name and the name of the first action are the same.
# These names are used for the step resource type.
STEP_NAME="%s"
STEP_ID=$(curl --silent -G --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects/${PROJECT_ID}/runbookProcesses/RunbookProcess-${RESOURCE_ID}" | jq -r ".Steps[] | select(.Name == \"${STEP_NAME}\") | .Id")

if [[ -z "${STEP_ID}" ]]
then
	echo "No step found with the name ${STEP_NAME}"
	exit 1
fi

echo "Importing runbook \"${RESOURCE_NAME}\" deployment process step \"${STEP_NAME}\" ${STEP_ID}"

# Step ID is in the format "RunbookProcess-Runbooks-123:00000000-0000-0000-0000-000000000001"
ID="%s.%s"
terraform state list "${ID}" &> /dev/null
if [[ $? -ne 0 ]]
then
	terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" "${ID}" RunbookProcess-${RESOURCE_ID}:${STEP_ID}
fi`,
					resourceName,
					resourceName,
					resourceName,
					resourceName,
					resourceName,
					projectName,
					runbookName,
					stepName,
					octopusdeployProcessStepResourceType,
					resourceName),
				nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c *RunbookProcessConverter) toStepPowershellImport(resourceName string, projectName string, runbookName string, stepName string, dependencies *data.ResourceDetailsCollection) {
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

$ProjectName="%s"

$headers = @{
    "X-Octopus-ApiKey" = $ApiKey
}

$ProjectId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ProjectName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ProjectName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ProjectId)) {
	echo "No project found with the name $ProjectName"
	exit 1
}

$ResourceName="%s"

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ProjectId/Runbooks?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No runbook found with the name $ResourceName"
	exit 1
}

$StepName="%s"

$StepId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ProjectId/deploymentprocesses" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Steps |
	Where-Object {$_.Name -eq $StepName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($StepId)) {
	echo "No step found with the name $StepName"
	exit 1
}

echo "Importing runbook $ResourceName deployment process step $StepName $StepId"

$Id="%s.%s"
terraform state list "${ID}" *> $null
if ($LASTEXITCODE -ne 0) {
	terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" $Id "RunbookProcess-$($ResourceId):$($StepId)"
}`,
					resourceName,
					projectName,
					runbookName,
					stepName,
					octopusdeployProcessStepResourceType,
					resourceName),
				nil
		},
	})
}

// toChildStepBashImport creates a bash script to import the child step resource
func (c *RunbookProcessConverter) toChildStepBashImport(resourceName string, projectName string, runbookName string, parentStepName string, stepName string, dependencies *data.ResourceDetailsCollection) {
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

if [[ -z "${PROJECT_ID}" ]]
then
	echo "No project found with the name ${PROJECT_NAME}"
	exit 1
fi

RESOURCE_NAME="%s"
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects/${PROJECT_ID}/Runbooks" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z "${RESOURCE_ID}" ]]
then
	echo "No project found with the name ${RESOURCE_NAME}"
	exit 1
fi

PARENT_STEP_NAME="%s"
PARENT_STEP_ID=$(curl --silent -G --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects/${PROJECT_ID}/runbookProcesses/RunbookProcess-${RESOURCE_ID}" | jq -r ".Steps[] | select(.Name == \"${PARENT_STEP_NAME}\") | .Id")

if [[ -z "${PARENT_STEP_ID}" ]]
then
	echo "No runbook parent step found with the name ${PARENT_STEP_NAME}"
	exit 1
fi

CHILD_STEP_NAME="%s"
CHILD_STEP_ID=$(curl --silent -G --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects/${PROJECT_ID}/runbookProcesses/RunbookProcess-${RESOURCE_ID}" | jq -r ".Steps[].Actions[] | select(.Name == \"${CHILD_STEP_NAME}\") | .Id")

if [[ -z "${CHILD_STEP_ID}" ]]
then
	echo "No child step found with the name ${CHILD_STEP_NAME}"
	exit 1
fi

echo "Importing runbook \"${RESOURCE_NAME}\" deployment process child step \"${CHILD_STEP_NAME}\" ${CHILD_STEP_ID}"

# Step ID is in the format "RunbookProcess-Runbooks-123:00000000-0000-0000-0000-000000000001"
ID="%s.%s"
terraform state list "${ID}" &> /dev/null
if [[ $? -ne 0 ]]
then
	terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" "${ID}" RunbookProcess-${RESOURCE_ID}:${PARENT_STEP_ID}:${CHILD_STEP_ID}
fi`,
					resourceName, // comments
					resourceName, // comments
					resourceName, // comments
					resourceName, // comments
					resourceName, // comments
					projectName,
					runbookName,
					parentStepName,
					stepName,
					octopusdeployProcessChildStepResourceType,
					resourceName),
				nil
		},
	})
}

// toChildStepPowershellImport creates a powershell script to import the child step resource
func (c *RunbookProcessConverter) toChildStepPowershellImport(resourceName string, projectName string, runbookName string, parentStepName string, stepName string, dependencies *data.ResourceDetailsCollection) {
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
$ProjectName="%s"

$headers = @{
    "X-Octopus-ApiKey" = $ApiKey
}

$ProjectId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ProjectName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ProjectName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ProjectId)) {
	echo "No project found with the name $ProjectName"
	exit 1
}

$ResourceName="%s"

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ProjectId/Runbooks?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No runbook found with the name $ResourceName"
	exit 1
}

$ParentStepName="%s"

$ParentStepId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ProjectId/deploymentprocesses" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Steps |
	Where-Object {$_.Name -eq $ParentStepName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ParentStepId)) {
	echo "No step found with the name $ParentStepName"
	exit 1
}

$ChildStepName="%s"

$ChildStepId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ProjectId/deploymentprocesses" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Steps | 
	Select-Object -ExpandProperty Actions | 
	Where-Object {$_.Name -eq $ChildStepName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ChildStepId)) {
	echo "No step found with the name $ChildStepName"
	exit 1
}

echo "Importing project $StepId"

$Id="%s.%s"
terraform state list "${ID}" *> $null
if ($LASTEXITCODE -ne 0) {
	terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" $Id "RunbookProcess-$($ResourceId):$($ParentStepId):$($ChildStepId)"
}`,
					resourceName,
					resourceName,
					projectName,
					runbookName,
					parentStepName,
					stepName,
					octopusdeployProcessChildStepResourceType,
					resourceName),
				nil
		},
	})
}
