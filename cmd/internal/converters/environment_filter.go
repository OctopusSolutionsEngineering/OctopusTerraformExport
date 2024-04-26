package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
)

// EnvironmentFilter provides a convenient way to get a list of environments taking into account any filtering
// options that may have been defined via the cli arguments.
type EnvironmentFilter struct {
	Client                              client.OctopusClient
	ExcludeVariableEnvironmentScopes    args.StringSliceArgs
	excludeVariableEnvironmentScopesIds []string
}

func (c EnvironmentFilter) convertEnvironmentsToIds() {
	if c.ExcludeVariableEnvironmentScopes == nil {
		c.excludeVariableEnvironmentScopesIds = []string{}
	} else {
		c.excludeVariableEnvironmentScopesIds = lo.FilterMap(c.ExcludeVariableEnvironmentScopes, func(envName string, index int) (string, bool) {

			// for each input environment name, convert it to an ID
			environments := octopus.GeneralCollection[octopus.Environment]{}
			err := c.Client.GetAllResources("Environments", &environments)
			if err == nil {
				// partial matches can have false positives, so do a second filter to do an exact match
				filteredList := lo.FilterMap(environments.Items, func(env octopus.Environment, index int) (string, bool) {
					if env.Name == envName {
						return env.Id, true
					}

					return "", false
				})

				// return the environment id
				if len(filteredList) != 0 {
					return filteredList[0], true
				}
			}

			// no match found
			return "", false
		})
	}
}

func (c EnvironmentFilter) FilterEnvironmentScope(envs []string) []string {
	if envs == nil {
		return []string{}
	}

	// One time conversion of the environment names to IDs
	if len(c.excludeVariableEnvironmentScopesIds) != len(c.ExcludeVariableEnvironmentScopes) {
		c.convertEnvironmentsToIds()
	}

	return lo.Filter(envs, func(env string, i int) bool {
		if c.excludeVariableEnvironmentScopesIds != nil && slices.Index(c.excludeVariableEnvironmentScopesIds, env) != -1 {
			return false
		}

		return true
	})
}
