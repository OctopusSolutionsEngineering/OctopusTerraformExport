package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type TenantProjectConverter struct {
	IncludeSpaceInPopulation bool
	ErrGroup                 *errgroup.Group
	ExcludeTenantTagSets     args.StringSliceArgs
	ExcludeTenantTags        args.StringSliceArgs
	ExcludeTenants           args.StringSliceArgs
	ExcludeTenantsRegex      args.StringSliceArgs
	ExcludeTenantsWithTags   args.StringSliceArgs
	ExcludeTenantsExcept     args.StringSliceArgs
	ExcludeAllTenants        bool
	ExcludeProjects          args.StringSliceArgs
	ExcludeProjectsExcept    args.StringSliceArgs
	ExcludeProjectsRegex     args.StringSliceArgs
	ExcludeAllProjects       bool
	Excluder                 ExcludeByName
	Client                   client.OctopusClient
}

func (c TenantProjectConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {

}

func (c *TenantProjectConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c TenantProjectConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTenants {
		return nil
	}

	collection := octopus.GeneralCollection[octopus.Tenant]{}
	err := c.Client.GetAllResources("Tenants", &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		for projectId, environmentId := range resource.ProjectEnvironments {
			project := octopus.Project{}
			_, err = c.Client.GetSpaceResourceById("Projects", projectId, &project)

			if err != nil {
				return err
			}

			if c.Excluder.IsResourceExcludedWithRegex(project.Name, c.ExcludeAllProjects, c.ExcludeProjects, c.ExcludeProjectsRegex, c.ExcludeProjectsExcept) {
				continue
			}

			zap.L().Info("Tenant: " + resource.Id + " Project: " + projectId)

			c.LinkTenantToProject(resource, project, environmentId, dependencies)
		}
	}

	return nil
}

func (c TenantProjectConverter) LinkTenantToProject(tenant octopus.Tenant, project octopus.Project, environmentIds []string, dependencies *data.ResourceDetailsCollection) {
	// Ignore excluded tenants
	if c.Excluder.IsResourceExcludedWithRegex(tenant.Name, c.ExcludeAllTenants, c.ExcludeTenants, c.ExcludeTenantsRegex, c.ExcludeTenantsExcept) {
		return
	}

	// Ignore tenants with excluded tags
	if c.ExcludeTenantsWithTags != nil && tenant.TenantTags != nil && lo.SomeBy(tenant.TenantTags, func(item string) bool {
		return lo.IndexOf(c.ExcludeTenantsWithTags, item) != -1
	}) {
		return
	}

	resourceName := "tenant_project_" + sanitizer.SanitizeName(tenant.Name) + "_" + sanitizer.SanitizeName(project.Name)

	tenantProject := data.ResourceDetails{}
	tenantProject.FileName = "space_population/" + resourceName + ".tf"
	tenantProject.Id = tenant.Id + "_" + project.Id
	tenantProject.Name = tenant.Name + " " + project.Name
	tenantProject.ResourceType = "TenantProject"
	tenantProject.Lookup = "${" + octopusdeployTenantProjectResourceType + "." + resourceName + ".id}"
	tenantProject.Dependency = "${" + octopusdeployTenantProjectResourceType + "." + resourceName + "}"

	tenantProject.ToHcl = func() (string, error) {
		terraformProjectEnvironments := terraform.TerraformTenantProjectEnvironment{
			Type:           octopusdeployTenantProjectResourceType,
			Name:           resourceName,
			Count:          nil,
			SpaceId:        strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", tenant.SpaceId)),
			TenantId:       dependencies.GetResource("Tenants", tenant.Id),
			ProjectId:      dependencies.GetResource("Projects", project.Id),
			EnvironmentIds: dependencies.GetResources("Environments", environmentIds...),
		}

		file := hclwrite.NewEmptyFile()
		projectEnvironmentBlock := gohcl.EncodeAsBlock(terraformProjectEnvironments, "resource")
		file.Body().AppendBlock(projectEnvironmentBlock)
		return string(file.Bytes()), nil
	}

	dependencies.AddResource(tenantProject)
}
