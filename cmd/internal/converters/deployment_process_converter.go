package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"go.uber.org/zap"
	"net/url"
)

// DeploymentProcessConverter converts deployment processes for v1 of the Octopus Terraform provider.
type DeploymentProcessConverter struct {
	DeploymentProcessConverterBase
}

func (c *DeploymentProcessConverter) ToHclByIdAndBranch(parentId string, branch string, recursive bool, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndBranch(parentId, branch, recursive, false, dependencies)
}

func (c *DeploymentProcessConverter) ToHclStatelessByIdAndBranch(parentId string, branch string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndBranch(parentId, branch, true, true, dependencies)
}

func (c *DeploymentProcessConverter) toHclByIdAndBranch(parentId string, branch string, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if parentId == "" || branch == "" {
		return nil
	}

	if dependencies.HasResource(parentId+"/"+branch, c.GetResourceType()) {
		return nil
	}

	// Get the deployment process associated with the git branch
	resource := octopus.DeploymentProcess{}
	found, err := c.Client.GetResource("Projects/"+parentId+"/"+url.QueryEscape(branch)+"/deploymentprocesses", &resource)

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

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", resource.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	c.exportScripts(project, resource, dependencies)

	return c.toHcl(&resource, nil, &project, recursive, false, stateless, dependencies)
}

func (c *DeploymentProcessConverter) ToHclLookupByIdAndBranch(parentId string, branch string, dependencies *data.ResourceDetailsCollection) error {
	if parentId == "" || branch == "" {
		return nil
	}

	if dependencies.HasResource(parentId+"/"+branch, c.GetResourceType()) {
		return nil
	}

	// Get the deployment process associated with the git branch
	resource := octopus.DeploymentProcess{}
	found, err := c.Client.GetResource("Projects/"+parentId+"/"+url.QueryEscape(branch)+"/deploymentprocesses", &resource)

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

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", resource.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	c.exportScripts(project, resource, dependencies)

	return c.toHcl(&resource, nil, &project, false, true, false, dependencies)
}

func (c *DeploymentProcessConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, false, dependencies)
}

func (c *DeploymentProcessConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, true, dependencies)
}

func (c *DeploymentProcessConverter) toHclById(id string, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.DeploymentProcess{}
	found, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.DeploymentProcess: %w", err)
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", resource.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	zap.L().Info("Deployment Process: " + resource.Id)

	c.exportScripts(project, resource, dependencies)
	return c.toHcl(&resource, nil, &project, recursive, false, stateless, dependencies)
}

func (c *DeploymentProcessConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.DeploymentProcess{}
	found, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.DeploymentProcess: %w", err)
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", resource.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	c.exportScripts(project, resource, dependencies)
	return c.toHcl(&resource, nil, &project, false, true, false, dependencies)
}

func (c *DeploymentProcessConverter) exportScripts(project octopus.Project, resource octopus.DeploymentProcess, dependencies *data.ResourceDetailsCollection) {
	if c.GenerateImportScripts {
		c.toBashImport(c.generateProcessName(nil, &project), c.generateStepOrderName(nil, &project), project.GetName(), dependencies)
		c.toPowershellImport(c.generateProcessName(nil, &project), c.generateStepOrderName(nil, &project), project.GetName(), dependencies)

		validSteps := c.getValidSteps(&resource)

		for _, step := range validSteps {
			c.toStepBashImport(
				c.generateStepName(nil, &project, &step),
				c.generateChildStepOrderName(nil, &project, &step),
				project.GetName(),
				step.GetName(),
				dependencies)
			c.toStepPowershellImport(
				c.generateStepName(nil, &project, &step),
				c.generateChildStepOrderName(nil, &project, &step),
				project.GetName(),
				step.GetName(),
				dependencies)

			for _, action := range step.Actions[1:] {
				c.toChildStepBashImport(
					c.generateChildStepName(nil, &project, &action),
					project.GetName(),
					step.GetName(),
					action.GetName(),
					dependencies)
				c.toChildStepPowershellImport(
					c.generateChildStepName(nil, &project, &action),
					project.GetName(),
					step.GetName(),
					action.GetName(),
					dependencies)
			}
		}
	}
}

// toBashImport creates a bash script to import the resource
func (c *DeploymentProcessConverter) toBashImport(resourceName string, stepsOrderResourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z "${RESOURCE_ID}" ]]
then
	echo "No project found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing project deployment process ${RESOURCE_NAME} ${RESOURCE_ID}"

ID="%s.%s"
terraform state list "${ID}" &> /dev/null
if [[ $? -ne 0 ]]
then
	terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" "${ID}" deploymentprocess-${RESOURCE_ID}
fi

ID="%s.%s"
terraform state list "${ID}" &> /dev/null
if [[ $? -ne 0 ]]
then
	terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" "${ID}" deploymentprocess-${RESOURCE_ID}
fi`,
					resourceName,
					resourceName,
					resourceName,
					resourceName,
					resourceName,
					projectName,
					octopusdeployProcessResourceType,
					resourceName,
					octopusdeployProcessStepsOrderResourceType,
					stepsOrderResourceName),
				nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c *DeploymentProcessConverter) toPowershellImport(resourceName string, stepsOrderResourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	Write-Error "No project found with the name $ResourceName"
	exit 1
}

echo "Importing project $ResourceId"

$Id="%s.%s"
terraform state list "${ID}" *> $null
if ($LASTEXITCODE -ne 0) {
	terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" $id deploymentprocess-$ResourceId
}

$Id="%s.%s"
terraform state list "${ID}" *> $null
if ($LASTEXITCODE -ne 0) {
	terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" $Id deploymentprocess-$ResourceId
}`,
					resourceName,
					projectName,
					octopusdeployProcessResourceType,
					resourceName,
					octopusdeployProcessStepsOrderResourceType,
					stepsOrderResourceName),
				nil
		},
	})
}

// toStepBashImport creates a bash script to import the step resource
func (c *DeploymentProcessConverterBase) toStepBashImport(resourceName string, childStepsOrderResourceName string, projectName string, stepName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z "${RESOURCE_ID}" ]]
then
	echo "No project found with the name ${RESOURCE_NAME}"
	exit 1
fi

# The step name and the name of the first action are the same.
# These names are used for the step resource type.
STEP_NAME="%s"
DEPLOYMENT_PROCESS=$(curl --silent -G --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects/${RESOURCE_ID}/deploymentprocesses")
STEP_ID=$(echo ${DEPLOYMENT_PROCESS} | jq -r ".Steps[] | select(.Name == \"${STEP_NAME}\") | .Id")
ACTION_COUNT=$(echo ${DEPLOYMENT_PROCESS} | jq -r ".Steps[] | select(.Name == \"${STEP_NAME}\") | .Actions | length")

if [[ -z "${STEP_ID}" ]]
then
	echo "No step found with the name ${STEP_NAME}"
	exit 1
fi

echo "Importing project deployment process step \"${STEP_NAME}\" ${STEP_ID}"

# Step ID is in the format "deploymentprocess-Projects-123:00000000-0000-0000-0000-000000000001"
ID="%s.%s"
terraform state list "${ID}" &> /dev/null
if [[ $? -ne 0 ]]
then
	terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" "${ID}" deploymentprocess-${RESOURCE_ID}:${STEP_ID}
fi

if [[ $ACTION_COUNT -gt 1 ]]
then
	ID="%s.%s"
	terraform state list "${ID}" &> /dev/null
	if [[ $? -ne 0 ]]
	then
		terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" "${ID}" deploymentprocess-${RESOURCE_ID}:${STEP_ID}
	fi
fi`,
					resourceName,
					resourceName,
					resourceName,
					resourceName,
					resourceName,
					projectName,
					stepName,
					octopusdeployProcessStepResourceType,
					resourceName,
					octopusdeployProcessChildStepsOrder,
					childStepsOrderResourceName),
				nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c *DeploymentProcessConverterBase) toStepPowershellImport(resourceName string, childStepsOrderResourceName string, projectName string, stepName string, dependencies *data.ResourceDetailsCollection) {
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
$StepName="%s"

$headers = @{
    "X-Octopus-ApiKey" = $ApiKey
}

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	Write-Error "No project found with the name $ResourceName"
	exit 1
}

$DeploymentProcess = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ResourceId/deploymentprocesses" -Method Get -Headers $headers
$StepId = $DeploymentProcess  |
	Select-Object -ExpandProperty Steps |
	Where-Object {$_.Name -eq $StepName} | 
	Select-Object -ExpandProperty Id
$ActionCount = $DeploymentProcess |
	Select-Object -ExpandProperty Steps |
	Where-Object {$_.Name -eq $StepName} |
	Select-Object -ExpandProperty Actions |
	Measure-Object |
	Select-Object -ExpandProperty Count

if ([System.String]::IsNullOrEmpty($StepId)) {
	Write-Error "No step found with the name $StepName"
	exit 1
}

echo "Importing project step $StepId for project $ResourceId"

$Id="%s.%s"
terraform state list "${ID}" *> $null
if ($LASTEXITCODE -ne 0) {
	terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" $Id "deploymentprocess-$($ResourceId):$($StepId)"
}

if ($ActionCount -gt 1) {
	$Id="%s.%s"
	terraform state list "${ID}" *> $null
	if ($LASTEXITCODE -ne 0) {
		terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" $Id "deploymentprocess-$($ResourceId):$($StepId)"
	}
}`,
					resourceName,
					projectName,
					stepName,
					octopusdeployProcessStepResourceType,
					resourceName,
					octopusdeployProcessChildStepsOrder,
					childStepsOrderResourceName),
				nil
		},
	})
}

// toChildStepBashImport creates a bash script to import the child step resource
func (c *DeploymentProcessConverterBase) toChildStepBashImport(resourceName string, projectName string, parentStepName string, stepName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z "${RESOURCE_ID}" ]]
then
	echo "No project found with the name ${RESOURCE_NAME}" >&2
	exit 1
fi

PARENT_STEP_NAME="%s"
PARENT_STEP_ID=$(curl --silent -G --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects/${RESOURCE_ID}/deploymentprocesses" | jq -r ".Steps[] | select(.Name == \"${PARENT_STEP_NAME}\") | .Id")

if [[ -z "${PARENT_STEP_ID}" ]]
then
	echo "No project parent step found with the name ${PARENT_STEP_NAME}" >&2
	exit 1
fi

CHILD_STEP_NAME="%s"
CHILD_STEP_ID=$(curl --silent -G --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects/${RESOURCE_ID}/deploymentprocesses" | jq -r ".Steps[].Actions[] | select(.Name == \"${CHILD_STEP_NAME}\") | .Id")

if [[ -z "${CHILD_STEP_ID}" ]]
then
	echo "No project child step found with the name ${CHILD_STEP_NAME}" >&2
	exit 1
fi

echo "Importing project deployment process child step \"${CHILD_STEP_NAME}\" ${CHILD_STEP_ID}"

# Step ID is in the format "deploymentprocess-Projects-123:00000000-0000-0000-0000-000000000001"
ID="%s.%s"
terraform state list "${ID}" &> /dev/null
if [[ $? -ne 0 ]]
then
	terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" "${ID}" deploymentprocess-${RESOURCE_ID}:${PARENT_STEP_ID}:${CHILD_STEP_ID}
fi`,
					resourceName,
					resourceName,
					resourceName,
					resourceName,
					resourceName,
					projectName,
					parentStepName,
					stepName,
					octopusdeployProcessChildStepResourceType,
					resourceName),
				nil
		},
	})
}

// toChildStepPowershellImport creates a powershell script to import the child step resource
func (c *DeploymentProcessConverterBase) toChildStepPowershellImport(resourceName string, projectName string, parentStepName string, stepName string, dependencies *data.ResourceDetailsCollection) {
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
$ParentStepName="%s"
$ChildStepName="%s"

$headers = @{
    "X-Octopus-ApiKey" = $ApiKey
}

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	Write-Error "No project found with the name $ResourceName"
	exit 1
}

$ParentStepId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ResourceId/deploymentprocesses" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Steps |
	Where-Object {$_.Name -eq $ParentStepName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ParentStepId)) {
	Write-Error "No step found with the name $ParentStepName"
	exit 1
}

$ChildStepId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ResourceId/deploymentprocesses" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Steps | 
	Select-Object -ExpandProperty Actions | 
	Where-Object {$_.Name -eq $ChildStepName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ChildStepId)) {
	Write-Error "No step found with the name $ChildStepName"
	exit 1
}

echo "Importing project child step $StepId into parent step $ParentStepId for project $ResourceId"

$Id="%s.%s"
terraform state list "${ID}" *> $null
if ($LASTEXITCODE -ne 0) {
	terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" $Id "deploymentprocess-$($ResourceId):$($ParentStepId):$($ChildStepId)"
}`,
					resourceName,
					projectName,
					parentStepName,
					stepName,
					octopusdeployProcessChildStepResourceType,
					resourceName),
				nil
		},
	})
}
