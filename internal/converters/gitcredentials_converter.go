package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type GitCredentialsConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
}

func (c GitCredentialsConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.GitCredentials]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}

	for _, gitCredentials := range collection.Items {
		gitCredentialsName := "gitcredential_" + util.SanitizeName(gitCredentials.Name)

		terraformResource := terraform.TerraformGitCredentials{
			Type:         "octopusdeploy_git_credential",
			Name:         gitCredentialsName,
			Description:  util.NilIfEmptyPointer(gitCredentials.Description),
			ResourceName: gitCredentials.Name,
			ResourceType: gitCredentials.Details.Type,
			Username:     gitCredentials.Details.Username,
			Password:     "${var." + gitCredentialsName + "}",
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		secretVariableResource := terraform.TerraformVariable{
			Name:        gitCredentialsName,
			Type:        "string",
			Nullable:    false,
			Sensitive:   true,
			Description: "The secret variable value associated with the git credential \"" + gitCredentials.Name + "\"",
		}

		block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
		util.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		results["space_population/gitcredentails_"+gitCredentialsName+".tf"] = string(file.Bytes())
		resultsMap[gitCredentials.Id] = "${octopusdeploy_tag_set." + gitCredentialsName + ".id}"
	}

	return results, resultsMap, nil
}

func (c GitCredentialsConverter) GetResourceType() string {
	return "Git-Credentials"
}
