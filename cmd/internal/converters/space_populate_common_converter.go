package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
)

// TerraformProviderGenerator creates the common terraform files required to populate a space
// including the provider, terraform config, and common vars
type TerraformProviderGenerator struct {
	TerraformBackend         string
	ProviderVersion          string
	ExcludeProvider          bool
	IncludeOctopusOutputVars bool
}

func (c TerraformProviderGenerator) ToHcl(directory string, includeSpaceId bool, dependencies *data.ResourceDetailsCollection) {
	c.createProvider(directory, includeSpaceId, dependencies)
	c.createTerraformConfig(directory, dependencies)
	c.createVariables(directory, includeSpaceId, dependencies)
	if c.IncludeOctopusOutputVars {
		c.createOctopusOutputVars(directory, includeSpaceId, dependencies)
	}
}

func (c TerraformProviderGenerator) createProvider(directory string, includeSpaceId bool, dependencies *data.ResourceDetailsCollection) {
	if c.ExcludeProvider {
		return
	}

	thisResource := data.ResourceDetails{}
	thisResource.FileName = directory + "/provider.tf"
	thisResource.Id = ""
	thisResource.ResourceType = ""
	thisResource.Lookup = ""
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform2.TerraformProvider{
			Type:    "octopusdeploy",
			Address: "${trimspace(var.octopus_server)}",
			ApiKey:  "${trimspace(var.octopus_apikey)}",
		}
		if includeSpaceId {
			spaceId := "${trimspace(var.octopus_space_id)}"
			terraformResource.SpaceId = &spaceId
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "provider"))

		shellScriptProvider := terraform2.TerraformShellProvider{
			Type:              "shell",
			Interpreter:       []string{"pwsh", "-Command"},
			EnableParallelism: false,
		}
		file.Body().AppendBlock(gohcl.EncodeAsBlock(shellScriptProvider, "provider"))

		externalProvider := terraform2.TerraformEmptyProvider{
			Type: "external",
		}
		file.Body().AppendBlock(gohcl.EncodeAsBlock(externalProvider, "provider"))

		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)
}

func (c TerraformProviderGenerator) createTerraformConfig(directory string, dependencies *data.ResourceDetailsCollection) {

	// When creating a module, we need to define the required providers, but not the backend
	backend := ""
	if !c.ExcludeProvider {
		backend = c.TerraformBackend
	}

	thisResource := data.ResourceDetails{}
	thisResource.FileName = directory + "/config.tf"
	thisResource.Id = ""
	thisResource.ResourceType = ""
	thisResource.Lookup = ""
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform2.TerraformConfig{}.CreateTerraformConfig(backend, c.ProviderVersion)
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "terraform"))
		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)
}

func (c TerraformProviderGenerator) createVariables(directory string, includeSpaceId bool, dependencies *data.ResourceDetailsCollection) {
	if c.ExcludeProvider {
		return
	}

	thisResource := data.ResourceDetails{}
	thisResource.FileName = directory + "/provider_vars.tf"
	thisResource.Id = ""
	thisResource.ResourceType = ""
	thisResource.Lookup = ""
	thisResource.ToHcl = func() (string, error) {
		octopusServer := terraform2.TerraformVariable{
			Name:        "octopus_server",
			Type:        "string",
			Nullable:    false,
			Sensitive:   false,
			Description: "The URL of the Octopus server e.g. https://myinstance.octopus.app.",
		}

		octopusApiKey := terraform2.TerraformVariable{
			Name:        "octopus_apikey",
			Type:        "string",
			Nullable:    false,
			Sensitive:   true,
			Description: "The API key used to access the Octopus server. See https://octopus.com/docs/octopus-rest-api/how-to-create-an-api-key for details on creating an API key.",
		}

		file := hclwrite.NewEmptyFile()

		octopusServerBlock := gohcl.EncodeAsBlock(octopusServer, "variable")
		hcl.WriteUnquotedAttribute(octopusServerBlock, "type", "string")
		file.Body().AppendBlock(octopusServerBlock)

		octopusApiKeyBlock := gohcl.EncodeAsBlock(octopusApiKey, "variable")
		hcl.WriteUnquotedAttribute(octopusApiKeyBlock, "type", "string")
		file.Body().AppendBlock(octopusApiKeyBlock)

		if includeSpaceId {
			octopusSpaceId := terraform2.TerraformVariable{
				Name:        "octopus_space_id",
				Type:        "string",
				Nullable:    false,
				Sensitive:   false,
				Description: "The ID of the Octopus space to populate.",
			}

			octopusSpaceIdBlock := gohcl.EncodeAsBlock(octopusSpaceId, "variable")
			hcl.WriteUnquotedAttribute(octopusSpaceIdBlock, "type", "string")
			file.Body().AppendBlock(octopusSpaceIdBlock)
		}

		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)
}

// createOctopusOutputVars captures the details of the octopus server as output variables. This is
// useful when finding the created resources from the Terraform state.
func (c TerraformProviderGenerator) createOctopusOutputVars(directory string, includeSpaceId bool, dependencies *data.ResourceDetailsCollection) {
	if c.ExcludeProvider {
		return
	}

	thisResource := data.ResourceDetails{}
	thisResource.FileName = directory + "/provider_output_vars.tf"
	thisResource.Id = ""
	thisResource.ResourceType = ""
	thisResource.Lookup = ""
	thisResource.ToHcl = func() (string, error) {
		octopusServer := terraform2.TerraformOutput{
			Name:  "octopus_server",
			Value: "${var.octopus_server}",
		}

		file := hclwrite.NewEmptyFile()

		octopusServerBlock := gohcl.EncodeAsBlock(octopusServer, "output")
		file.Body().AppendBlock(octopusServerBlock)

		if includeSpaceId {
			octopusSpaceId := terraform2.TerraformOutput{
				Name:  "octopus_space_id",
				Value: "${var.octopus_space_id}",
			}

			octopusSpaceNameData := terraform2.TerraformSpaceData{
				Type:         "octopusdeploy_spaces",
				Name:         "octopus_space_name",
				ResourceName: nil,
				Ids:          []string{"${var.octopus_space_id}"},
				PartialName:  nil,
				Skip:         0,
				Take:         1,
			}

			octopusSpaceName := terraform2.TerraformOutput{
				Name:  "octopus_space_name",
				Value: "${data.octopusdeploy_spaces.octopus_space_name.spaces[0].name}",
			}

			octopusSpaceIdBlock := gohcl.EncodeAsBlock(octopusSpaceId, "output")
			file.Body().AppendBlock(octopusSpaceIdBlock)

			file.Body().AppendBlock(gohcl.EncodeAsBlock(octopusSpaceNameData, "data"))
			file.Body().AppendBlock(gohcl.EncodeAsBlock(octopusSpaceName, "output"))
		}

		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)
}
