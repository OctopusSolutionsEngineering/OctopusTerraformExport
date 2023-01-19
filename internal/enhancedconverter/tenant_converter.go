package enhancedconverter

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type TenantConverter struct {
	Client client.OctopusClient
}

func (c TenantConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Tenant]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		err = c.toHcl(resource, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c TenantConverter) ToHclByProjectId(projectId string, dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Tenant]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection, []string{"projectId", projectId})

	if err != nil {
		return nil
	}

	for _, tenant := range collection.Items {
		err = c.toHcl(tenant, dependencies)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (c TenantConverter) toHcl(tenant octopus.Tenant, dependencies *ResourceDetailsCollection) error {

	// Export all the tag sets
	err := TagSetConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Export the tenant variables
	err = TenantVariableConverter{
		Client: c.Client,
	}.ToHclByTenantId(tenant.Id, dependencies)

	if err != nil {
		return err
	}

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

func (c TenantConverter) GetResourceType() string {
	return "Tenants"
}

func (c TenantConverter) getProjects(tags map[string][]string, dependencies *ResourceDetailsCollection) []terraform.TerraformProjectEnvironment {
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

func (c TenantConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}
