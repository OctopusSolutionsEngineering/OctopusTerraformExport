package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

const octopusdeployGitCredentialDataType = "octopusdeploy_git_credentials"
const octopusdeployGitCredentialResourceType = "octopusdeploy_git_credential"

type GitCredentialsConverter struct {
	Client                    client.OctopusClient
	SpaceResourceName         string
	DummySecretVariableValues bool
	DummySecretGenerator      DummySecretGenerator
}

func (c GitCredentialsConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.GitCredentials]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Git Credentials: " + resource.Id)
		err = c.toHcl(resource, false, false, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c GitCredentialsConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.GitCredentials{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Git Credentials: " + resource.Id)
	return c.toHcl(resource, true, false, false, dependencies)
}

func (c GitCredentialsConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	gitCredentials := octopus2.GitCredentials{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &gitCredentials)

	if err != nil {
		return err
	}

	return c.toHcl(gitCredentials, false, true, false, dependencies)
}

func (c GitCredentialsConverter) toHcl(gitCredentials octopus2.GitCredentials, _ bool, lookup bool, stateless bool, dependencies *ResourceDetailsCollection) error {

	gitCredentialsName := "gitcredential_" + sanitizer.SanitizeName(gitCredentials.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + gitCredentialsName + ".tf"
	thisResource.Id = gitCredentials.Id
	thisResource.ResourceType = c.GetResourceType()

	if lookup {
		c.toHclLookup(gitCredentials, &thisResource, gitCredentialsName)
	} else {
		c.toHclResource(stateless, gitCredentials, &thisResource, gitCredentialsName)
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c GitCredentialsConverter) toHclLookup(gitCredentials octopus2.GitCredentials, thisResource *ResourceDetails, gitCredentialsName string) {
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

func (c GitCredentialsConverter) buildData(resourceName string, resource octopus2.GitCredentials) terraform2.TerraformGitCredentialData {
	return terraform2.TerraformGitCredentialData{
		Type:         octopusdeployGitCredentialDataType,
		Name:         resourceName,
		ResourceName: resource.Name,
		Skip:         0,
		Take:         1,
	}
}

// writeData appends the data block for stateless modules
func (c GitCredentialsConverter) writeData(file *hclwrite.File, resource octopus2.GitCredentials, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c GitCredentialsConverter) toHclResource(stateless bool, gitCredentials octopus2.GitCredentials, thisResource *ResourceDetails, gitCredentialsName string) {
	thisResource.Lookup = "${" + octopusdeployGitCredentialResourceType + "." + gitCredentialsName + ".id}"

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployGitCredentialDataType + "." + gitCredentialsName + ".git_credentials) != 0 " +
			"? data." + octopusdeployGitCredentialDataType + "." + gitCredentialsName + ".git_credentials[0].id " +
			": " + octopusdeployGitCredentialResourceType + "." + gitCredentialsName + "[0].id}"
	} else {
		thisResource.Lookup = "${" + octopusdeployGitCredentialResourceType + "." + gitCredentialsName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform2.TerraformGitCredentials{
			Type:         octopusdeployGitCredentialResourceType,
			Name:         gitCredentialsName,
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

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), gitCredentials.Name, octopusdeployGitCredentialResourceType, gitCredentialsName))

		targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		// When using dummy values, we expect the secrets will be updated later
		if c.DummySecretVariableValues {
			hcl.WriteLifecycleAttribute(targetBlock, "[password]")
		}

		file.Body().AppendBlock(targetBlock)

		secretVariableResource := terraform2.TerraformVariable{
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
