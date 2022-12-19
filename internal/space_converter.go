package internal

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
)

type spaceTerraform struct {
	Type                     string   `hcl:"type,label"`
	Name                     string   `hcl:"name,label"`
	Description              string   `hcl:"description"`
	ResourceName             string   `hcl:"name"`
	IsDefault                bool     `hcl:"is_default"`
	IsTaskQueueStopped       bool     `hcl:"is_task_queue_stopped"`
	SpaceManagersTeamMembers []string `hcl:"space_managers_team_members"`
	SpaceManagersTeams       []string `hcl:"space_managers_teams"`
}

type SpaceConverter struct {
	Client OctopusClient
}

func (c SpaceConverter) ToHcl() (string, error) {
	space := Space{}
	err := c.Client.GetSpace(&space)

	if err != nil {
		return "", err
	}

	terraformResource := spaceTerraform{
		Description:              space.Description,
		IsDefault:                space.IsDefault,
		IsTaskQueueStopped:       space.TaskQueueStopped,
		ResourceName:             "octopus_space_" + space.Id,
		SpaceManagersTeamMembers: space.SpaceManagersTeamMembers,
		SpaceManagersTeams:       space.SpaceManagersTeams,
		Name:                     space.Name,
		Type:                     "octopusdeploy_space",
	}
	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
	return string(file.Bytes()), nil
}

func (c SpaceConverter) getResourceType() string {
	return "Spaces"
}
