package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

// SpaceConverter creates the files required to create a new space. These files are used in a separate
// terraform project, as you first need to a create a space, and then configure a second provider
// to use that space.
type SpaceConverter struct {
	Client client.OctopusClient
}

func (c SpaceConverter) ToHcl() (map[string]string, error) {

	spaceTf, err := c.createSpaceTf()

	if err != nil {
		return nil, err
	}

	provider := c.createSpaceProvider()
	terraformConfig := c.createTerraformConfig()
	terraformVariables := c.createVariables()

	results := map[string]string{
		internal.CreateSpaceDir + "/space.tf":    spaceTf,
		internal.CreateSpaceDir + "/provider.tf": provider,
		internal.CreateSpaceDir + "/config.tf":   terraformConfig,
		internal.CreateSpaceDir + "/vars.tf":     terraformVariables,
	}

	// Generate space population common files
	commonProjectFiles := SpacePopulateCommonGenerator{}.ToHcl()

	// merge the maps
	for k, v := range commonProjectFiles {
		results[k] = v
	}

	// Convert the projects
	projects, err := c.processProjects()
	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range projects {
		results[k] = v
	}

	return results, nil
}

func (c SpaceConverter) getResourceType() string {
	return "Spaces"
}

func (c SpaceConverter) processProjects() (map[string]string, error) {
	return ProjectConverter{
		Client: c.Client,
	}.ToHcl()
}

func (c SpaceConverter) createSpaceTf() (string, error) {
	space := model.Space{}
	err := c.Client.GetSpace(&space)

	if err != nil {
		return "", err
	}

	terraformResource := model.TerraformSpace{
		Description:              space.Description,
		IsDefault:                space.IsDefault,
		IsTaskQueueStopped:       space.TaskQueueStopped,
		Name:                     "octopus_space_" + util.SanitizeName(space.Name),
		SpaceManagersTeamMembers: space.SpaceManagersTeamMembers,
		SpaceManagersTeams:       space.SpaceManagersTeams,
		ResourceName:             space.Name,
		Type:                     "octopusdeploy_space",
	}

	spaceOutput := model.TerraformOutput{
		Name:  "octopus_space_id",
		Value: "octopusdeploy_space.octopus_space_" + util.SanitizeName(space.Name) + ".id",
	}

	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
	file.Body().AppendBlock(gohcl.EncodeAsBlock(spaceOutput, "output"))
	return string(file.Bytes()), nil
}

func (c SpaceConverter) createSpaceProvider() string {
	terraformResource := model.TerraformProvider{
		Type:    "octopusdeploy",
		Address: "var.octopus_server",
		ApiKey:  "var.octopus_apikey",
	}
	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "provider"))
	return string(file.Bytes())
}

func (c SpaceConverter) createTerraformConfig() string {
	terraformResource := model.TerraformConfig{}.CreateTerraformConfig()
	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "terraform"))
	return string(file.Bytes())
}

func (c SpaceConverter) createVariables() string {
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

	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(octopusServer, "variable"))
	file.Body().AppendBlock(gohcl.EncodeAsBlock(octopusApiKey, "variable"))
	return string(file.Bytes())
}
