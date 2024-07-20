package converters

import (
	"errors"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

const octopusdeployTenantProjectVariableResourceType = "octopusdeploy_tenant_project_variable"

type TenantVariableConverter struct {
	Client                         client.OctopusClient
	ExcludeTenants                 args.StringSliceArgs
	ExcludeTenantsWithTags         args.StringSliceArgs
	ExcludeTenantsExcept           args.StringSliceArgs
	ExcludeAllTenants              bool
	Excluder                       ExcludeByName
	DummySecretVariableValues      bool
	DummySecretGenerator           DummySecretGenerator
	ExcludeProjects                args.StringSliceArgs
	ExcludeProjectsExcept          args.StringSliceArgs
	ExcludeProjectsRegex           args.StringSliceArgs
	ExcludeAllProjects             bool
	ErrGroup                       *errgroup.Group
	ExcludeAllTenantVariables      bool
	ExcludeTenantVariables         args.StringSliceArgs
	ExcludeTenantVariablesExcept   args.StringSliceArgs
	ExcludeTenantVariablesRegex    args.StringSliceArgs
	TenantCommonVariableProcessor  TenantCommonVariableProcessor
	TenantProjectVariableConverter TenantProjectVariableConverter
}

func (c TenantVariableConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c TenantVariableConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c TenantVariableConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	collection := []octopus.TenantVariable{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection {
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c TenantVariableConverter) ToHclByTenantId(id string, dependencies *data.ResourceDetailsCollection) error {
	resource := octopus.TenantVariable{}
	err := c.Client.GetAllResources("Tenants/"+id+"/Variables", &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, true, false, dependencies)
}

// ToHclByTenantIdAndProject is used by projects to export tenant variables relating to the project and any
// library variable sets referenced by the project.
// Tenant variables are a resource that don't fit nicely into the split between space level resources and
// project level resources.
// Tenant project variables have a clear dependency between a tenant and a project.
// Tenant common variables have an implicit (and often hard to reason about) dependency between a tenant, project, and the
// library variable set referenced by the project.
// This means it is up to the project to define any tenant variables relating to the project, as these variables can
// only be created once the project is available.
func (c TenantVariableConverter) ToHclByTenantIdAndProject(id string, project octopus.Project, dependencies *data.ResourceDetailsCollection) error {
	resource := octopus.TenantVariable{}
	err := c.Client.GetAllResources("Tenants/"+id+"/Variables", &resource)

	if err != nil {
		return err
	}

	// only include library variables referenced by the project
	resource.LibraryVariables = lo.PickBy(resource.LibraryVariables, func(key string, value octopus.LibraryVariable) bool {
		return lo.Contains(project.IncludedLibraryVariableSetIds, value.LibraryVariableSetId)
	})

	// only include project variables for the project
	resource.ProjectVariables = lo.PickBy(resource.ProjectVariables, func(key string, value octopus.ProjectVariable) bool {
		return value.ProjectId == project.Id
	})

	return c.toHcl(resource, true, false, dependencies)
}

func (c TenantVariableConverter) toHcl(tenant octopus.TenantVariable, _ bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {

	// Ignore excluded tenants
	if c.Excluder.IsResourceExcluded(tenant.TenantName, c.ExcludeAllTenants, c.ExcludeTenants, c.ExcludeTenantsExcept) {
		return nil
	}

	// Ignore tenants with excluded tags
	excluded, err := c.isTenantExcludedByTag(tenant.TenantId)

	if err != nil {
		return err
	}

	if excluded {
		return nil
	}

	if err2 := c.convertProjectVariables(tenant, stateless, dependencies); err2 != nil {
		return err2
	}

	if err3 := c.convertCommonVariables(tenant, stateless, dependencies); err3 != nil {
		return err3
	}

	return nil
}

func (c TenantVariableConverter) convertCommonVariables(tenant octopus.TenantVariable, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	for _, l := range tenant.LibraryVariables {
		commonVariableIndex := 0

		for id, value := range l.Variables {

			libraryVariableSet := octopus.LibraryVariableSet{}
			_, err := c.Client.GetSpaceResourceById("LibraryVariableSets", l.LibraryVariableSetId, &libraryVariableSet)

			if err != nil {
				return err
			}

			commonVariableIndex++

			if err := c.TenantCommonVariableProcessor.ConvertTenantCommonVariable(
				stateless, tenant, id, value, libraryVariableSet, commonVariableIndex, dependencies); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c TenantVariableConverter) convertProjectVariables(tenant octopus.TenantVariable, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Don't attempt to link variables from excluded projects
	var filterErr error = nil
	filteredProjectVariables := lo.Filter(lo.Values[string, octopus.ProjectVariable](tenant.ProjectVariables), func(item octopus.ProjectVariable, index int) bool {
		varExcluded, varExcludedErr := c.excludeProject(item.ProjectId)
		if varExcludedErr != nil {
			filterErr = errors.Join(filterErr, varExcludedErr)
			return false
		}

		return !varExcluded
	})

	if filterErr != nil {
		return filterErr
	}

	for _, projectVariable := range filteredProjectVariables {

		projectVariableIndex := 0

		for environmentId, variable := range projectVariable.Variables {
			for templateId, value := range variable {
				value := value

				projectVariableIndex++
				if err := c.TenantProjectVariableConverter.ConvertTenantProjectVariable(
					stateless, tenant, projectVariable, environmentId, value, projectVariableIndex, templateId, dependencies); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c TenantVariableConverter) GetResourceType() string {
	return "TenantVariables/All"
}

func (c TenantVariableConverter) isTenantExcludedByTag(tenantId string) (bool, error) {
	// Ignore tenants with excluded tags
	resource := octopus.Tenant{}
	found, err := c.Client.GetSpaceResourceById("Tenants", tenantId, &resource)

	if err != nil {
		return false, err
	}

	if found && resource.TenantTags != nil && c.ExcludeTenantsWithTags != nil {
		return lo.SomeBy(resource.TenantTags, func(item string) bool {
			return lo.IndexOf(c.ExcludeTenantsWithTags, item) != -1
		}), nil
	}

	return false, nil
}

func (c TenantVariableConverter) excludeProject(projectId string) (bool, error) {
	if c.ExcludeAllProjects {
		return true, nil
	}

	project := octopus.Project{}
	_, err := c.Client.GetSpaceResourceById("Projects", projectId, &project)

	if err != nil {
		return false, err
	}

	return c.Excluder.IsResourceExcludedWithRegex(project.Name, c.ExcludeAllProjects, c.ExcludeProjects, c.ExcludeProjectsRegex, c.ExcludeProjectsExcept), nil
}
