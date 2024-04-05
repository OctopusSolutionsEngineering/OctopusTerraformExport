package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
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

	collection := octopus2.GeneralCollection[octopus2.GitCredentials]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Git Credentials: " + resource.Id)
		err = c.toHcl(resource, false, false, stateless, dependencies)

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

	resource := octopus2.GitCredentials{}
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

	gitCredentials := octopus2.GitCredentials{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &gitCredentials)

	if err != nil {
		return err
	}

	return c.toHcl(gitCredentials, false, true, false, dependencies)
}

func (c GitCredentialsConverter) toHcl(gitCredentials octopus2.GitCredentials, _ bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {

	if c.ExcludeAllGitCredentials {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + gitCredentials.Id)
		return nil
	}

	gitCredentialsName := "gitcredential_" + sanitizer.SanitizeName(gitCredentials.Name)

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

func (c GitCredentialsConverter) toHclLookup(gitCredentials octopus2.GitCredentials, thisResource *data.ResourceDetails, gitCredentialsName string) {
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

func (c GitCredentialsConverter) toHclResource(stateless bool, gitCredentials octopus2.GitCredentials, dependencies *data.ResourceDetailsCollection, thisResource *data.ResourceDetails, gitCredentialsName string) {
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

		terraformResource := terraform2.TerraformGitCredentials{
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

			ignoreAll := terraform2.EmptyBlock{}
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
