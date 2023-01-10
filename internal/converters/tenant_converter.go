package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type TenantConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	EnvironmentsMap   map[string]string
	ProjectsMap       map[string]string
}

func (c TenantConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.Tenant]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}

	for _, tenant := range collection.Items {
		tenantName := "tenant_" + util.SanitizeName(tenant.Name)

		terraformResource := terraform.TerraformTenant{
			Type:               "octopusdeploy_tenant",
			Name:               tenantName,
			ResourceName:       tenant.Name,
			Id:                 nil,
			ClonedFromTenantId: nil,
			Description:        util.NilIfEmptyPointer(tenant.Description),
			TenantTags:         tenant.TenantTags,
			ProjectEnvironment: c.getProjects(tenant.ProjectEnvironments),
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		results["space_population/tenant_"+tenantName+".tf"] = string(file.Bytes())
		resultsMap[tenant.Id] = "${octopusdeploy_tenant." + tenantName + ".id}"
	}

	return results, resultsMap, nil
}

func (c TenantConverter) GetResourceType() string {
	return "Tenants"
}

func (c TenantConverter) getProjects(tags map[string][]string) []terraform.TerraformProjectEnvironment {
	terraformProjectEnvironments := make([]terraform.TerraformProjectEnvironment, len(tags))
	index := 0
	for k, v := range tags {
		terraformProjectEnvironments[index] = terraform.TerraformProjectEnvironment{
			Environments: c.lookupEnvironments(v),
			ProjectId:    c.ProjectsMap[k],
		}
		index++
	}
	return terraformProjectEnvironments
}

func (c TenantConverter) lookupEnvironments(envs []string) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = c.EnvironmentsMap[v]
	}
	return newEnvs
}
