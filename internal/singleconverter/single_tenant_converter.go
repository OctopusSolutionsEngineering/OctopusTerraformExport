package singleconverter

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type SingleTenantConverter struct {
	Client client.OctopusClient
}

func (c SingleTenantConverter) ToHcl(projectId string, dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Tenant]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection, []string{"projectId", projectId})

	if err != nil {
		return nil
	}

	for _, tenant := range collection.Items {
		err = c.toHcl(tenant, projectId, dependencies)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (c SingleTenantConverter) toHcl(tenant octopus.Tenant, projectId string, dependencies *ResourceDetailsCollection) error {
	tenantName := "tenant_" + util.SanitizeName(tenant.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + tenantName + ".tf"
	thisResource.Id = tenant.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_tenant." + tenantName + ".id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformTenant{
			Type:               "octopusdeploy_tenant",
			Name:               tenantName,
			ResourceName:       tenant.Name,
			Id:                 nil,
			ClonedFromTenantId: nil,
			Description:        util.NilIfEmptyPointer(tenant.Description),
			TenantTags:         tenant.TenantTags,
			ProjectEnvironment: c.getProjects(tenant.ProjectEnvironments, dependencies),
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c SingleTenantConverter) GetResourceType() string {
	return "Tenants"
}

func (c SingleTenantConverter) getProjects(tags map[string][]string, dependencies *ResourceDetailsCollection) []terraform.TerraformProjectEnvironment {
	terraformProjectEnvironments := make([]terraform.TerraformProjectEnvironment, len(tags))
	index := 0
	for k, v := range tags {
		terraformProjectEnvironments[index] = terraform.TerraformProjectEnvironment{
			Environments: c.lookupEnvironments(v, dependencies),
			ProjectId:    dependencies.GetResource("Projects", k),
		}
		index++
	}
	return terraformProjectEnvironments
}

func (c SingleTenantConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}
