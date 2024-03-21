package converters

import (
	"errors"
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

const octopusdeployTenantProjectVariableResourceType = "octopusdeploy_tenant_project_variable"

type TenantVariableConverter struct {
	Client                       client.OctopusClient
	ExcludeTenants               args.StringSliceArgs
	ExcludeTenantsWithTags       args.StringSliceArgs
	ExcludeTenantsExcept         args.StringSliceArgs
	ExcludeAllTenants            bool
	Excluder                     ExcludeByName
	DummySecretVariableValues    bool
	DummySecretGenerator         DummySecretGenerator
	ExcludeProjects              args.StringSliceArgs
	ExcludeProjectsExcept        args.StringSliceArgs
	ExcludeProjectsRegex         args.StringSliceArgs
	ExcludeAllProjects           bool
	ErrGroup                     *errgroup.Group
	ExcludeAllTenantVariables    bool
	ExcludeTenantVariables       args.StringSliceArgs
	ExcludeTenantVariablesExcept args.StringSliceArgs
	ExcludeTenantVariablesRegex  args.StringSliceArgs
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

	// Assume the tenant has added the data block to resolve existing tenants. Use that data block
	// to test if any of the tenant variables should be created.
	tenantName := "tenant_" + sanitizer.SanitizeName(tenant.TenantName)
	var count *string = nil
	if stateless {
		count = strutil.StrPointer("${length(data." + octopusdeployTenantsDataType + "." + tenantName + ".tenants) != 0 ? 0 : 1}")
	}

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

	for _, p := range filteredProjectVariables {

		projectVariableIndex := 0

		for env, variable := range p.Variables {
			for templateId, value := range variable {
				value := value

				projectVariableIndex++
				variableName := "tenantprojectvariable_" + fmt.Sprint(projectVariableIndex) + "_" + sanitizer.SanitizeName(tenant.TenantName)

				thisResource := data.ResourceDetails{}
				thisResource.FileName = "space_population/" + variableName + ".tf"
				thisResource.Id = templateId
				thisResource.ResourceType = c.GetResourceType()
				thisResource.Lookup = "${" + octopusdeployTenantProjectVariableResourceType + "." + variableName + ".id}"

				if stateless {
					thisResource.Lookup = "${length(data." + octopusdeployTenantsDataType + "." + tenantName + ".tenants) != 0 " +
						"? '' " +
						": " + octopusdeployTenantProjectVariableResourceType + "." + variableName + "[0].id}"
					thisResource.Dependency = "${" + octopusdeployTenantProjectVariableResourceType + "." + variableName + "}"
				} else {
					thisResource.Lookup = "${" + octopusdeployTenantProjectVariableResourceType + "." + variableName + ".id}"
				}

				thisResource.ToHcl = func() (string, error) {
					file := hclwrite.NewEmptyFile()

					terraformResource := terraform2.TerraformTenantProjectVariable{
						Type:          octopusdeployTenantProjectVariableResourceType,
						Name:          variableName,
						Count:         count,
						Id:            nil,
						EnvironmentId: dependencies.GetResource("Environments", env),
						ProjectId:     dependencies.GetResource("Projects", p.ProjectId),
						TemplateId:    dependencies.GetResource("ProjectTemplates", templateId),
						TenantId:      dependencies.GetResource("Tenants", tenant.TenantId),
						Value:         &value,
					}

					block := gohcl.EncodeAsBlock(terraformResource, "resource")

					if stateless {
						hcl.WriteLifecyclePreventDestroyAttribute(block)
					}

					file.Body().AppendBlock(block)
					return string(file.Bytes()), nil
				}
				dependencies.AddResource(thisResource)
			}
		}
	}

	for _, l := range tenant.LibraryVariables {
		commonVariableIndex := 0

		for id, value := range l.Variables {

			libraryVariableSet := octopus.LibraryVariableSet{}
			c.Client.GetResourceById("LibraryVariableSets", l.LibraryVariableSetId, &libraryVariableSet)
			libraryVariableSetVariableName := lo.Filter(libraryVariableSet.Templates, func(item octopus.Template, index int) bool {
				return item.Id == id
			})

			if len(libraryVariableSetVariableName) != 0 {
				// Do not export excluded variables
				if c.Excluder.IsResourceExcludedWithRegex(strutil.EmptyIfNil(libraryVariableSetVariableName[0].Name),
					c.ExcludeAllTenantVariables,
					c.ExcludeTenantVariables,
					c.ExcludeTenantVariablesRegex,
					c.ExcludeTenantVariablesExcept) {
					continue
				}
			}

			commonVariableIndex++
			variableName := "tenantcommonvariable" + fmt.Sprint(commonVariableIndex) + "_" + sanitizer.SanitizeName(tenant.TenantName)

			thisResource := data.ResourceDetails{}
			thisResource.FileName = "space_population/" + variableName + ".tf"
			thisResource.Id = id
			thisResource.ResourceType = c.GetResourceType()
			thisResource.Lookup = "${octopusdeploy_tenant_common_variable." + variableName + ".id}"

			/*
				Tenants can define secrets, in which case value is an object indicating the state of the
				secret, but not the value. In this case we can only export an empty string.
				TODO: Create a variable to override this value if needed.
			*/
			fixedValue := ""
			if stringValue, ok := value.(string); ok {
				fixedValue = stringValue
			}

			l := l
			id := id
			tenant := tenant

			thisResource.ToHcl = func() (string, error) {
				file := hclwrite.NewEmptyFile()
				terraformResource := terraform2.TerraformTenantCommonVariable{
					Type:                 "octopusdeploy_tenant_common_variable",
					Name:                 variableName,
					Count:                count,
					Id:                   nil,
					LibraryVariableSetId: dependencies.GetResource("LibraryVariableSets", l.LibraryVariableSetId),
					TemplateId:           dependencies.GetResource("CommonTemplateMap", id),
					TenantId:             dependencies.GetResource("Tenants", tenant.TenantId),
					Value:                &fixedValue,
				}
				file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
				return string(file.Bytes()), nil
			}
			dependencies.AddResource(thisResource)
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
	found, err := c.Client.GetResourceById("Tenants", tenantId, &resource)

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

func (c *TenantVariableConverter) excludeProject(projectId string) (bool, error) {
	if c.ExcludeAllProjects {
		return true, nil
	}

	project := octopus.Project{}
	_, err := c.Client.GetResourceById("Projects", projectId, &project)

	if err != nil {
		return false, err
	}

	return c.Excluder.IsResourceExcludedWithRegex(project.Name, c.ExcludeAllProjects, c.ExcludeProjects, c.ExcludeProjectsRegex, c.ExcludeProjectsExcept), nil
}
