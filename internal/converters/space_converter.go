package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
	"strings"
)

// SpaceConverter creates the files required to create a new space. These files are used in a separate
// terraform project, as you first need to a create a space, and then configure a second provider
// to use that space.
type SpaceConverter struct {
	Client client.OctopusClient
}

func (c SpaceConverter) ToHcl() (map[string]string, error) {

	spaceResourceName, spaceTf, err := c.createSpaceTf()

	if err != nil {
		return nil, err
	}

	results := map[string]string{
		"space.tf": spaceTf,
	}

	// Generate space population common files
	commonProjectFiles := SpacePopulateCommonGenerator{}.ToHcl()

	// merge the maps
	for k, v := range commonProjectFiles {
		results[k] = v
	}

	// Convert the feeds
	feeds, feedMap, err := FeedConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range feeds {
		results[k] = v
	}

	// Convert the accounts
	accounts, accountsMap, err := AccountConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range accounts {
		results[k] = v
	}

	// Convert the lifecycles
	lifecycles, lifecycleMap, err := LifecycleConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range lifecycles {
		results[k] = v
	}

	// Convert the worker pools
	pools, poolMap, err := WorkerPoolConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range pools {
		results[k] = v
	}

	// Convert the projects groups
	projects, err := ProjectGroupConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
		FeedMap:           feedMap,
		LifecycleMap:      lifecycleMap,
		WorkPoolMap:       poolMap,
		AccountsMap:       accountsMap,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range projects {
		results[k] = v
	}

	// Unescape dollar signs because of https://github.com/hashicorp/hcl/issues/323
	for k, v := range results {
		results[k] = strings.ReplaceAll(v, "$${", "${")
	}

	return results, nil
}

func (c SpaceConverter) getResourceType() string {
	return "Spaces"
}

func (c SpaceConverter) createSpaceTf() (string, string, error) {
	space := octopus.Space{}
	err := c.Client.GetSpace(&space)

	if err != nil {
		return "", "", err
	}

	spaceResourceName := "octopus_space_" + util.SanitizeName(space.Name)
	spaceName := "${var." + spaceResourceName + "_name}"

	terraformResource := terraform.TerraformSpace{
		Description:        space.Description,
		IsDefault:          space.IsDefault,
		IsTaskQueueStopped: space.TaskQueueStopped,
		Name:               spaceResourceName,
		//SpaceManagersTeamMembers: space.SpaceManagersTeamMembers,
		//SpaceManagersTeams:       space.SpaceManagersTeams,
		// TODO: import teams rather than defaulting to admins
		SpaceManagersTeams: []string{"teams-administrators"},
		ResourceName:       &spaceName,
		Type:               "octopusdeploy_space",
	}

	spaceOutput := terraform.TerraformOutput{
		Name:  "octopus_space_id",
		Value: "${octopusdeploy_space." + spaceResourceName + ".id}",
	}

	spaceNameVar := terraform.TerraformVariable{
		Name:        spaceResourceName + "_name",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The name of the new space (the exported space was called " + *space.Name + ")",
	}

	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
	file.Body().AppendBlock(gohcl.EncodeAsBlock(spaceOutput, "output"))

	block := gohcl.EncodeAsBlock(spaceNameVar, "variable")
	util.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	return spaceResourceName, string(file.Bytes()), nil
}
