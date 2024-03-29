package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"k8s.io/utils/strings/slices"
	"strings"
)

const octopusdeployTenantsDataType = "octopusdeploy_tenants"
const octopusdeployTenantResourceType = "octopusdeploy_tenant"

type TenantConverter struct {
	Client                  client.OctopusClient
	TenantVariableConverter ConverterByTenantId
	EnvironmentConverter    ConverterAndLookupWithStatelessById
	TagSetConverter         ConvertToHclByResource[octopus2.TagSet]
	ExcludeTenantTagSets    args.StringSliceArgs
	ExcludeTenantTags       args.StringSliceArgs
	ExcludeTenants          args.StringSliceArgs
	ExcludeTenantsRegex     args.StringSliceArgs
	ExcludeTenantsWithTags  args.StringSliceArgs
	ExcludeTenantsExcept    args.StringSliceArgs
	ExcludeAllTenants       bool
	Excluder                ExcludeByName
	ExcludeProjects         args.StringSliceArgs
	ExcludeProjectsExcept   args.StringSliceArgs
	ExcludeProjectsRegex    args.StringSliceArgs
	ExcludeAllProjects      bool
	ErrGroup                *errgroup.Group
	IncludeIds              bool
}

func (c *TenantConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c *TenantConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c *TenantConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTenants {
		return nil
	}

	collection := octopus2.GeneralCollection[octopus2.Tenant]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Tenant: " + resource.Id)
		err = c.toHcl(resource, false, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *TenantConverter) ToHclStatelessByProjectId(projectId string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByProjectId(projectId, true, dependencies)
}

func (c *TenantConverter) ToHclByProjectId(projectId string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByProjectId(projectId, false, dependencies)
}

func (c *TenantConverter) toHclByProjectId(projectId string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTenants {
		return nil
	}

	collection := octopus2.GeneralCollection[octopus2.Tenant]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection, []string{"projectId", projectId})

	if err != nil {
		return nil
	}

	for _, resource := range collection.Items {
		zap.L().Info("Tenant: " + resource.Id)
		err = c.toHcl(resource, true, false, stateless, dependencies)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (c *TenantConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTenants {
		return nil
	}

	resource := octopus2.Tenant{}
	found, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil
	}

	if found {
		zap.L().Info("Tenant: " + resource.Id)
		return c.toHcl(resource, true, false, false, dependencies)
	}

	return nil
}

func (c *TenantConverter) ToHclLookupByProjectId(projectId string, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTenants {
		return nil
	}

	collection := octopus2.GeneralCollection[octopus2.Tenant]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection, []string{"projectId", projectId})

	if err != nil {
		return nil
	}

	for _, tenant := range collection.Items {
		err = c.toHcl(tenant, false, true, false, dependencies)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (c *TenantConverter) buildData(resourceName string, resource octopus2.Tenant) terraform.TerraformTenantData {
	return terraform.TerraformTenantData{
		Type:        octopusdeployTenantsDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c *TenantConverter) writeData(file *hclwrite.File, resource octopus2.Tenant, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c *TenantConverter) toHcl(tenant octopus2.Tenant, recursive bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {

	// Ignore excluded tenants
	if c.Excluder.IsResourceExcludedWithRegex(tenant.Name, c.ExcludeAllTenants, c.ExcludeTenants, c.ExcludeTenantsRegex, c.ExcludeTenantsExcept) {
		return nil
	}

	// Ignore tenants with excluded tags
	if c.ExcludeTenantsWithTags != nil && tenant.TenantTags != nil && lo.SomeBy(tenant.TenantTags, func(item string) bool {
		return lo.IndexOf(c.ExcludeTenantsWithTags, item) != -1
	}) {
		return nil
	}

	if recursive {
		// Export the tenant variables
		err := c.TenantVariableConverter.ToHclByTenantId(tenant.Id, dependencies)

		if err != nil {
			return err
		}

		// Export the tenant environments
		for _, environments := range tenant.ProjectEnvironments {
			for _, environment := range environments {
				if stateless {
					err = c.EnvironmentConverter.ToHclStatelessById(environment, dependencies)
				} else {
					err = c.EnvironmentConverter.ToHclById(environment, dependencies)
				}
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

	tenantName := "tenant_" + sanitizer.SanitizeName(tenant.Name)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + tenantName + ".tf"
	thisResource.Id = tenant.Id
	thisResource.Name = tenant.Name
	thisResource.ResourceType = c.GetResourceType()

	if lookup {
		thisResource.Lookup = "${data." + octopusdeployTenantsDataType + "." + tenantName + ".tenants[0].id}"
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData(tenantName, tenant)
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a tenant called \""+tenant.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.tenants) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	} else {
		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployTenantsDataType + "." + tenantName + ".tenants) != 0 " +
				"? data." + octopusdeployTenantsDataType + "." + tenantName + ".tenants[0].id " +
				": " + octopusdeployTenantResourceType + "." + tenantName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployTenantResourceType + "." + tenantName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeployTenantResourceType + "." + tenantName + ".id}"
		}

		var count *string = nil
		if stateless {
			count = strutil.StrPointer("${length(data." + octopusdeployTenantsDataType + "." + tenantName + ".tenants) != 0 ? 0 : 1}")
		}

		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform.TerraformTenant{
				Type:               octopusdeployTenantResourceType,
				Name:               tenantName,
				Id:                 strutil.InputPointerIfEnabled(c.IncludeIds, &tenant.Id),
				Count:              count,
				ResourceName:       tenant.Name,
				ClonedFromTenantId: nil,
				Description:        strutil.NilIfEmptyPointer(tenant.Description),
				TenantTags:         c.Excluder.FilteredTenantTags(tenant.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			}

			projectEnvironments, err := c.getProjects(tenant.ProjectEnvironments, dependencies)

			if err != nil {
				return "", err
			}

			terraformResource.ProjectEnvironment = projectEnvironments

			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, tenant, tenantName)
			}

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			if stateless {
				hcl.WriteLifecyclePreventDestroyAttribute(block)
			}

			// Explicitly describe the dependency between a target and a tag set
			dependsOn := []string{}
			for resourceType, terraformDependencies := range tagSetDependencies {
				for _, terraformDependency := range terraformDependencies {
					dependency := dependencies.GetResourceDependency(resourceType, terraformDependency)
					dependency = hcl.RemoveId(hcl.RemoveInterpolation(dependency))
					if dependency != "" {
						dependsOn = append(dependsOn, dependency)
					}
				}
			}

			hcl.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(dependsOn[:], ",")+"]")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c *TenantConverter) GetResourceType() string {
	return "Tenants"
}

func (c *TenantConverter) excludeProject(projectId string) (bool, error) {
	if c.ExcludeAllProjects {
		return true, nil
	}

	project := octopus2.Project{}
	_, err := c.Client.GetResourceById("Projects", projectId, &project)

	if err != nil {
		return false, err
	}

	return c.Excluder.IsResourceExcludedWithRegex(project.Name, c.ExcludeAllProjects, c.ExcludeProjects, c.ExcludeProjectsRegex, c.ExcludeProjectsExcept), nil
}

func (c *TenantConverter) getProjects(tags map[string][]string, dependencies *data.ResourceDetailsCollection) ([]terraform.TerraformProjectEnvironment, error) {
	terraformProjectEnvironments := []terraform.TerraformProjectEnvironment{}
	for k, v := range tags {
		exclude, err := c.excludeProject(k)

		if err != nil {
			return []terraform.TerraformProjectEnvironment{}, err
		}

		if exclude {
			continue
		}

		projectId := dependencies.GetResource("Projects", k)

		// This shouldn't be empty, but test defensively anyway just in case.
		if projectId != "" {
			terraformProjectEnvironments = append(terraformProjectEnvironments, terraform.TerraformProjectEnvironment{
				Environments: c.lookupEnvironments(v, dependencies),
				ProjectId:    dependencies.GetResource("Projects", k),
			})
		}
	}
	return terraformProjectEnvironments, nil
}

func (c *TenantConverter) lookupEnvironments(envs []string, dependencies *data.ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

// addTagSetDependencies finds the tag sets that contains the tags associated with a tenant. These dependencies are
// captured, as Terraform has no other way to map the dependency between a tagset and a tenant.
func (c *TenantConverter) addTagSetDependencies(tenant octopus2.Tenant, recursive bool, dependencies *data.ResourceDetailsCollection) (map[string][]string, error) {
	collection := octopus2.GeneralCollection[octopus2.TagSet]{}
	err := c.Client.GetAllResources("TagSets", &collection)

	if err != nil {
		return nil, err
	}

	terraformDependencies := map[string][]string{}

	for _, tagSet := range collection.Items {
		if c.Excluder.IsResourceExcluded(tagSet.Name, false, c.ExcludeTenantTagSets, nil) {
			continue
		}

		for _, tag := range tagSet.Tags {

			if c.Excluder.IsResourceExcluded(tag.CanonicalTagName, false, c.ExcludeTenantTags, nil) {
				continue
			}

			for _, tenantTag := range tenant.TenantTags {
				if tag.CanonicalTagName == tenantTag {

					if !slices.Contains(terraformDependencies["TagSets"], tagSet.Id) {
						terraformDependencies["TagSets"] = append(terraformDependencies["TagSets"], tagSet.Id)
					}

					if !slices.Contains(terraformDependencies["Tags"], tag.Id) {
						terraformDependencies["Tags"] = append(terraformDependencies["Tags"], tag.Id)
					}

					if recursive {
						err = c.TagSetConverter.ToHclByResource(tagSet, dependencies)

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
