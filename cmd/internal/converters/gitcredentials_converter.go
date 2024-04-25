package converters

import (
	"fmt"
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

const octopusdeployGitCredentialDataType = "octopusdeploy_git_credentials"
const octopusdeployGitCredentialResourceType = "octopusdeploy_git_credential"

type GitCredentialsConverter struct {
	Client                    client.OctopusClient
	SpaceResourceName         string
	DummySecretVariableValues bool
	DummySecretGenerator      DummySecretGenerator
	ExcludeAllGitCredentials  bool
	ErrGroup                  *errgroup.Group
	IncludeIds                bool
	LimitResourceCount        int
	IncludeSpaceInPopulation  bool
	GenerateImportScripts     bool
}

func (c GitCredentialsConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c GitCredentialsConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c GitCredentialsConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllGitCredentials {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.GitCredentials]{
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

		zap.L().Info("Git Credentials: " + resource.Id)
		err := c.toHcl(resource, false, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c GitCredentialsConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c GitCredentialsConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c GitCredentialsConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllGitCredentials {
		return nil
	}

	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.GitCredentials{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Git Credentials: " + resource.Id)
	return c.toHcl(resource, true, false, stateless, dependencies)
}

func (c GitCredentialsConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllGitCredentials {
		return nil
	}

	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	gitCredentials := octopus.GitCredentials{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &gitCredentials)

	if err != nil {
		return err
	}

	return c.toHcl(gitCredentials, false, true, false, dependencies)
}

// toBashImport creates a bash script to import the resource
func (c GitCredentialsConverter) toBashImport(resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Git-Credentials" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No git credentials found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing git credentials ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, octopusdeployGitCredentialResourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c GitCredentialsConverter) toPowershellImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Git-Credentials?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No git credentials found with the name $ResourceName"
	exit 1
}

echo "Importing git credentials $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, octopusdeployGitCredentialResourceType, resourceName), nil
		},
	})
}

func (c GitCredentialsConverter) toHcl(gitCredentials octopus.GitCredentials, _ bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {

	if c.ExcludeAllGitCredentials {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + gitCredentials.Id)
		return nil
	}

	gitCredentialsName := "gitcredential_" + sanitizer.SanitizeName(gitCredentials.Name)

	if c.GenerateImportScripts {
		c.toBashImport(gitCredentialsName, gitCredentials.Name, dependencies)
		c.toPowershellImport(gitCredentialsName, gitCredentials.Name, dependencies)
	}

	thisResource := data.ResourceDetails{}
	thisResource.Name = gitCredentials.Name
	thisResource.FileName = "space_population/" + gitCredentialsName + ".tf"
	thisResource.Id = gitCredentials.Id
	thisResource.ResourceType = c.GetResourceType()

	if lookup {
		c.toHclLookup(gitCredentials, &thisResource, gitCredentialsName)
	} else {
		c.toHclResource(stateless, gitCredentials, dependencies, &thisResource, gitCredentialsName)
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c GitCredentialsConverter) toHclLookup(gitCredentials octopus.GitCredentials, thisResource *data.ResourceDetails, gitCredentialsName string) {
	thisResource.Lookup = "${data." + octopusdeployGitCredentialDataType + "." + gitCredentialsName + ".git_credentials[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(gitCredentialsName, gitCredentials)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a git credential called \""+gitCredentials.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.git_credentials) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
}

func (c GitCredentialsConverter) buildData(resourceName string, resource octopus.GitCredentials) terraform.TerraformGitCredentialData {
	return terraform.TerraformGitCredentialData{
		Type:         octopusdeployGitCredentialDataType,
		Name:         resourceName,
		ResourceName: resource.Name,
		Skip:         0,
		Take:         1,
	}
}

// writeData appends the data block for stateless modules
func (c GitCredentialsConverter) writeData(file *hclwrite.File, resource octopus.GitCredentials, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c GitCredentialsConverter) toHclResource(stateless bool, gitCredentials octopus.GitCredentials, dependencies *data.ResourceDetailsCollection, thisResource *data.ResourceDetails, gitCredentialsName string) {
	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployGitCredentialDataType + "." + gitCredentialsName + ".git_credentials) != 0 " +
			"? data." + octopusdeployGitCredentialDataType + "." + gitCredentialsName + ".git_credentials[0].id " +
			": " + octopusdeployGitCredentialResourceType + "." + gitCredentialsName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployGitCredentialResourceType + "." + gitCredentialsName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployGitCredentialResourceType + "." + gitCredentialsName + ".id}"
	}

	thisResource.Parameters = []data.ResourceParameter{
		{
			Label:         "Git Credentials " + gitCredentials.Name + " password",
			Description:   "The password associated with the feed \"" + gitCredentials.Name + "\"",
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, gitCredentials.Name, "Password"),
			ParameterType: "Password",
			Sensitive:     true,
			VariableName:  gitCredentialsName,
		},
	}
	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformGitCredentials{
			Type:         octopusdeployGitCredentialResourceType,
			Name:         gitCredentialsName,
			Id:           strutil.InputPointerIfEnabled(c.IncludeIds, &gitCredentials.Id),
			SpaceId:      strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", gitCredentials.SpaceId)),
			Description:  strutil.NilIfEmptyPointer(gitCredentials.Description),
			ResourceName: gitCredentials.Name,
			ResourceType: gitCredentials.Details.Type,
			Username:     gitCredentials.Details.Username,
			Password:     "${var." + gitCredentialsName + "}",
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, gitCredentials, gitCredentialsName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployGitCredentialDataType + "." + gitCredentialsName + ".git_credentials) != 0 ? 0 : 1}")
		}

		gitCertBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		// When using dummy values, we expect the secrets will be updated later
		if c.DummySecretVariableValues || stateless {

			ignoreAll := terraform.EmptyBlock{}
			lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
			gitCertBlock.Body().AppendBlock(lifecycleBlock)

			if c.DummySecretVariableValues {
				hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "[password]")
			}

			if stateless {
				hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
			}
		}

		file.Body().AppendBlock(gitCertBlock)

		secretVariableResource := terraform.TerraformVariable{
			Name:        gitCredentialsName,
			Type:        "string",
			Nullable:    false,
			Sensitive:   true,
			Description: "The secret variable value associated with the git credential \"" + gitCredentials.Name + "\"",
		}

		if c.DummySecretVariableValues {
			secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
		}

		block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
		hcl.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
}

func (c GitCredentialsConverter) GetResourceType() string {
	return "Git-Credentials"
}
