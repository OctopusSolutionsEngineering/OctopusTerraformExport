package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"golang.org/x/sync/errgroup"
)

type PlatformHubConverter struct {
	Client   client.OctopusClient
	ErrGroup *errgroup.Group
}

func (c PlatformHubConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c PlatformHubConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c PlatformHubConverter) allToHcl(all bool, dependencies *data.ResourceDetailsCollection) error {
	resource := &octopus.OctopusPlatformHubVersionControlUsernamePasswordSetting{}
	if err := c.Client.GetAllGlobalResources("PlatformHub/VersionControl", &resource, []string{}, []string{}); err != nil {
		return err
	}
	return nil
}
