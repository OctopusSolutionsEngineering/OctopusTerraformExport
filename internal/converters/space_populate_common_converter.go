package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

// SpacePopulateCommonGenerator creates the common terraform files required to populate a space
// including the provider, terraform config, and common vars
type SpacePopulateCommonGenerator struct {
	Client client.OctopusClient
}

func (c SpacePopulateCommonGenerator) ToHcl() map[string]string {
	provider := c.createProvider()
	terraformConfig := c.createTerraformConfig()
	terraformVariables := c.createVariables()

	return map[string]string{
		"provider.tf":      provider,
		"config.tf":        terraformConfig,
		"provider_vars.tf": terraformVariables,
	}
}

func (c SpacePopulateCommonGenerator) createProvider() string {
	spaceId := "${var.octopus_space_id}"
	terraformResource := terraform.TerraformProvider{
		Type:    "octopusdeploy",
		Address: "${var.octopus_server}",
		ApiKey:  "${var.octopus_apikey}",
		SpaceId: &spaceId,
	}
	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "provider"))
	return string(file.Bytes())
}

func (c SpacePopulateCommonGenerator) createTerraformConfig() string {
	terraformResource := terraform.TerraformConfig{}.CreateTerraformConfig()
	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "terraform"))
	return string(file.Bytes())
}

func (c SpacePopulateCommonGenerator) createVariables() string {
	octopusServer := terraform.TerraformVariable{
		Name:        "octopus_server",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The URL of the Octopus server e.g. https://myinstance.octopus.app.",
	}

	octopusApiKey := terraform.TerraformVariable{
		Name:        "octopus_apikey",
		Type:        "string",
		Nullable:    false,
		Sensitive:   true,
		Description: "The API key used to access the Octopus server. See https://octopus.com/docs/octopus-rest-api/how-to-create-an-api-key for details on creating an API key.",
	}

	octopusSpaceId := terraform.TerraformVariable{
		Name:        "octopus_space_id",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The ID of the Octopus space to populate.",
	}

	file := hclwrite.NewEmptyFile()

	octopusServerBlock := gohcl.EncodeAsBlock(octopusServer, "variable")
	util.WriteUnquotedAttribute(octopusServerBlock, "type", "string")
	file.Body().AppendBlock(octopusServerBlock)

	octopusApiKeyBlock := gohcl.EncodeAsBlock(octopusApiKey, "variable")
	util.WriteUnquotedAttribute(octopusApiKeyBlock, "type", "string")
	file.Body().AppendBlock(octopusApiKeyBlock)

	octopusSpaceIdBlock := gohcl.EncodeAsBlock(octopusSpaceId, "variable")
	util.WriteUnquotedAttribute(octopusSpaceIdBlock, "type", "string")
	file.Body().AppendBlock(octopusSpaceIdBlock)

	return string(file.Bytes())
}
