package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

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

	results := map[string]string{
		internal.CreateSpaceDir + "/space.tf":    spaceTf,
		internal.CreateSpaceDir + "/provider.tf": provider,
		internal.CreateSpaceDir + "/config.tf":   terraformConfig,
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
	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
	return string(file.Bytes()), nil
}

func (c SpaceConverter) createSpaceProvider() string {
	terraformResource := model.TerraformProvider{
		Type:    "octopusdeploy",
		Address: "var.octopusUrl",
		ApiKey:  "var.octopusApiKey",
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
