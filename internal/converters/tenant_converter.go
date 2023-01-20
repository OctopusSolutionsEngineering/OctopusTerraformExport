package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
	"k8s.io/utils/strings/slices"
	"strings"
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
		err = c.toHcl(resource, false, dependencies)

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
		err = c.toHcl(tenant, true, dependencies)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (c TenantConverter) toHcl(tenant octopus.Tenant, recursive bool, dependencies *ResourceDetailsCollection) error {

	if recursive {
		// Export the tenant variables
		err := TenantVariableConverter{
			Client: c.Client,
		}.ToHclByTenantId(tenant.Id, dependencies)

		if err != nil {
			return err
		}

		// Export the tenant environments
		for _, environments := range tenant.ProjectEnvironments {
			for _, environment := range environments {
				err = EnvironmentConverter{
					Client: c.Client,
				}.ToHclById(environment, dependencies)
			}
		}

		if err != nil {
			return err
		}
	}

	tagSetDependencies, err := c.addTagSetDependencies(tenant, recursive, dependencies)

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
		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		// Explicitly describe the dependency between a target and a tag set
		dependsOn := make([]string, len(tagSetDependencies))
		for i, t := range tagSetDependencies {
			dependency := dependencies.GetResource("TagSets", t)
			// This is a raw expression, so remove the surrounding brackets
			dependency = strings.Replace(dependency, "${", "", -1)
			dependency = strings.Replace(dependency, ".id}", "", -1)
			dependsOn[i] = dependency
		}

		util.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(dependsOn[:], ",")+"]")
		file.Body().AppendBlock(block)

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

// addTagSetDependencies finds the tag sets that contains the tags associated with a tenant. These dependencies are
// captured, as Terraform has no other way to map the dependency between a tagset and a tenant.
func (c TenantConverter) addTagSetDependencies(tenant octopus.Tenant, recursive bool, dependencies *ResourceDetailsCollection) ([]string, error) {
	collection := octopus.GeneralCollection[octopus.TagSet]{}
	err := c.Client.GetAllResources("TagSets", &collection)

	if err != nil {
		return nil, err
	}

	terraformDependencies := []string{}

	for _, tagSet := range collection.Items {
		for _, tag := range tagSet.Tags {
			for _, tenantTag := range tenant.TenantTags {
				if tag.CanonicalTagName == tenantTag {

					if !slices.Contains(terraformDependencies, tagSet.Id) {
						terraformDependencies = append(terraformDependencies, tagSet.Id)
					}

					if recursive {
						err = TagSetConverter{
							Client: c.Client,
						}.ToHclByResource(tagSet, recursive, dependencies)

						if err != nil {
							return nil, err
						}
					}
				}
			}
		}
	}

	return terraformDependencies, nil
}
