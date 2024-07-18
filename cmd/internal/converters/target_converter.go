package converters

import (
	"errors"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/samber/lo"
)

type TargetConverter struct {
	Client                           client.OctopusClient
	ExcludeEnvironments              args.StringSliceArgs
	ExcludeEnvironmentsRegex         args.StringSliceArgs
	ExcludeEnvironmentsExcept        args.StringSliceArgs
	ExcludeAllEnvironments           bool
	ExcludeTargetsWithNoEnvironments bool
	Excluder                         ExcludeByName
}

func (c TargetConverter) HasNoEnvironments(target octopus.TargetResource) (error, bool) {
	if c.ExcludeTargetsWithNoEnvironments {
		var exclusionError error = nil
		filteredEnvironments := lo.Filter(target.GetEnvironmentIds(), func(item string, index int) bool {
			environment := &octopus.Environment{}
			exists, err := c.Client.GetSpaceResourceById("Environments", item, environment)
			if err != nil {
				exclusionError = errors.Join(exclusionError, err)
				return false
			}

			if !exists {
				return false
			}

			excluded := c.Excluder.IsResourceExcludedWithRegex(
				environment.Name,
				c.ExcludeAllEnvironments,
				c.ExcludeEnvironments,
				c.ExcludeEnvironmentsRegex,
				c.ExcludeEnvironmentsExcept)

			return !excluded
		})

		if exclusionError != nil {
			return exclusionError, true
		}

		if len(filteredEnvironments) == 0 {
			return nil, true
		}
	}

	return nil, false
}
