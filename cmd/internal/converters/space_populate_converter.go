package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"golang.org/x/sync/errgroup"
)

// SpacePopulateConverter creates a space resource in the space population scripts. This has no value when
// creating Terraform modules to apply to live systems (and will likely break thins), and is only used when
// creating the context for a LLM query to indicate the space that the query relates to.
type SpacePopulateConverter struct {
	Client                   client.OctopusClient
	IncludeSpaceInPopulation bool
	IncludeIds               bool
	ErrGroup                 *errgroup.Group
}

// AllToHcl is a bulk export that takes advantage of the collection endpoints to download and export everything
// with no filter and with the least number of network calls.
func (c SpacePopulateConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {

	if c.IncludeSpaceInPopulation {
		c.ErrGroup.Go(func() error { return c.createSpaceTf(dependencies) })
	}
}

// AllToStatelessHcl is not used when exporting stateless resources
func (c SpacePopulateConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {

}

func (c SpacePopulateConverter) getResourceType() string {
	return "Spaces"
}

func (c SpacePopulateConverter) createSpaceTf(dependencies *data.ResourceDetailsCollection) error {
	space := octopus.Space{}
	err := c.Client.GetSpace(&space)

	if err != nil {
		return err
	}

	spaceResourceName := "octopus_space_" + sanitizer.SanitizeName(space.Name)
	spaceName := "${var.octopus_space_name}"

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + spaceResourceName + ".tf"
	thisResource.Id = space.Id
	thisResource.ResourceType = "Spaces"
	thisResource.Lookup = "${octopusdeploy_space." + spaceResourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform2.TerraformSpace{
			Description:        strutil.TrimPointer(space.Description),
			Id:                 strutil.InputPointerIfEnabled(c.IncludeIds, &space.Id),
			IsDefault:          space.IsDefault,
			IsTaskQueueStopped: space.TaskQueueStopped,
			Name:               spaceResourceName,
			SpaceManagersTeams: []string{"${var.octopus_space_managers}"},
			ResourceName:       &spaceName,
			Type:               "octopusdeploy_space",
		}
		file := hclwrite.NewEmptyFile()

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}
