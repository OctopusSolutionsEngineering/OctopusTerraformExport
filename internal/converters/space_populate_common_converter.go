package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model"
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
		internal.PopulateSpaceDir + "/provider.tf":      provider,
		internal.PopulateSpaceDir + "/config.tf":        terraformConfig,
		internal.PopulateSpaceDir + "/provider_vars.tf": terraformVariables,
	}
}

func (c SpacePopulateCommonGenerator) createProvider() string {
	terraformResource := model.TerraformProvider{
		Type:    "octopusdeploy",
		Address: "var.octopus_server",
		ApiKey:  "var.octopus_apikey",
		SpaceId: "var.octopus_space_id",
	}
	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "provider"))
	return string(file.Bytes())
}

func (c SpacePopulateCommonGenerator) createTerraformConfig() string {
	terraformResource := model.TerraformConfig{}.CreateTerraformConfig()
	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "terraform"))
	return string(file.Bytes())
}

func (c SpacePopulateCommonGenerator) createVariables() string {
	octopusServer := model.TerraformVariable{
		Name:        "octopus_server",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The URL of the Octopus server e.g. https://myinstance.octopus.app.",
	}

	octopusApiKey := model.TerraformVariable{
		Name:        "octopus_apikey",
		Type:        "string",
		Nullable:    false,
		Sensitive:   true,
		Description: "The API key used to access the Octopus server. See https://octopus.com/docs/octopus-rest-api/how-to-create-an-api-key for details on creating an API key.",
	}

	octopusSpaceId := model.TerraformVariable{
		Name:        "octopus_space_id",
		Type:        "string",
		Nullable:    false,
		Sensitive:   true,
		Description: "The Octopus space to populate.",
	}

	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(octopusServer, "variable"))
	file.Body().AppendBlock(gohcl.EncodeAsBlock(octopusApiKey, "variable"))
	file.Body().AppendBlock(gohcl.EncodeAsBlock(octopusSpaceId, "variable"))
	return string(file.Bytes())
}