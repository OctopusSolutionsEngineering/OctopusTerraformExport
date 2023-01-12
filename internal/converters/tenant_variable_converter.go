package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type TenantVariableConverter struct {
	Client                client.OctopusClient
	SpaceResourceName     string
	EnvironmentsMap       map[string]string
	ProjectsMap           map[string]string
	LibraryVariableSetMap map[string]string
	TenantsMap            map[string]string
	ProjectTemplatesMap   map[string]string
}

func (c TenantVariableConverter) ToHcl() (map[string]string, error) {
	collection := []octopus.TenantVariable{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, err
	}

	results := map[string]string{}

	projectVariableIndex := 0
	commonVariableIndex := 0
	for _, tenant := range collection {

		file := hclwrite.NewEmptyFile()
		for _, p := range tenant.ProjectVariables {
			for env, variable := range p.Variables {
				for templateId, value := range variable {
					projectVariableIndex++
					variableName := "tenantprojectvariable" + fmt.Sprint(projectVariableIndex) + "_" + util.SanitizeName(tenant.TenantName)
					terraformResource := terraform.TerraformTenantProjectVariable{
						Type:          "octopusdeploy_tenant_project_variable",
						Name:          variableName,
						Id:            nil,
						EnvironmentId: c.EnvironmentsMap[env],
						ProjectId:     c.ProjectsMap[p.ProjectId],
						TemplateId:    c.ProjectTemplatesMap[templateId],
						TenantId:      c.TenantsMap[tenant.TenantId],
						Value:         &value,
					}
					file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
				}
			}
		}

		for _, l := range tenant.LibraryVariables {
			for id, value := range l.Variables {
				commonVariableIndex++
				variableName := "tenantcommonvariable" + fmt.Sprint(commonVariableIndex) + "_" + util.SanitizeName(tenant.TenantName)
				terraformResource := terraform.TerraformTenantCommonVariable{
					Type:                 "octopusdeploy_tenant_common_variable",
					Name:                 variableName,
					Id:                   nil,
					LibraryVariableSetId: c.LibraryVariableSetMap[l.LibraryVariableSetId],
					TemplateId:           id,
					TenantId:             tenant.TenantId,
					Value:                &value,
				}
				file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
			}
		}

		tenantName := "tenantvariable_" + util.SanitizeName(tenant.TenantName)
		results["space_population/tenantvariable_"+tenantName+".tf"] = string(file.Bytes())

	}

	return results, nil
}

func (c TenantVariableConverter) GetResourceType() string {
	return "TenantVariables/All"
}
