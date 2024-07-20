package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
)

type TenantProjectConverter struct {
	IncludeSpaceInPopulation bool
}

func (c TenantProjectConverter) LinkTenantToProject(tenant octopus.Tenant, project octopus.Project, environmentIds []string, dependencies *data.ResourceDetailsCollection) {
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
