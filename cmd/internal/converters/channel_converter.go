package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"strings"
)

const octopusdeployChannelDataType = "octopusdeploy_channels"
const octopusdeployChannelResourceType = "octopusdeploy_channel"

type ChannelConverter struct {
	Client               client.OctopusClient
	LifecycleConverter   ConverterAndLookupById
	ExcludeTenantTags    args.ExcludeTenantTags
	ExcludeTenantTagSets args.ExcludeTenantTagSets
	Excluder             ExcludeByName
}

func (c ChannelConverter) ToHclByProjectIdWithTerraDependencies(projectId string, terraformDependencies map[string]string, dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Channel]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return err
	}

	project := octopus.Project{}
	_, err = c.Client.GetResourceById(c.GetResourceType(), projectId, &project)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Channel: " + resource.Id)
		project := octopus.Project{}
		err = c.toHcl(resource, project, true, false, false, terraformDependencies, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

// ToHclLookupByProjectIdWithTerraDependencies exports the channel set as a complete resource, but will reference external resources like
// lifecycles as data source lookups.
func (c ChannelConverter) ToHclLookupByProjectIdWithTerraDependencies(projectId string, terraformDependencies map[string]string, dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Channel]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return err
	}

	project := octopus.Project{}
	_, err = c.Client.GetResourceById(c.GetResourceType(), projectId, &project)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Channel: " + resource.Id)
		err = c.toHcl(resource, project, false, true, false, terraformDependencies, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

// We consider channels to be the responsibility of a project. If the project exists, we don't create the channel.
func (c ChannelConverter) buildData(resourceName string, name string) terraform.TerraformProjectData {
	return terraform.TerraformProjectData{
		Type:        octopusdeployProjectsDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c ChannelConverter) writeData(file *hclwrite.File, name string, resourceName string) {
	terraformResource := c.buildData(resourceName, name)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c ChannelConverter) toHcl(channel octopus.Channel, project octopus.Project, recursive bool, lookup bool, stateless bool, terraformDependencies map[string]string, dependencies *ResourceDetailsCollection) error {
	if channel.LifecycleId != "" {
		var err error
		if recursive {
			// The lifecycle is a dependency that we need to lookup
			err = c.LifecycleConverter.ToHclById(channel.LifecycleId, dependencies)
		} else if lookup {
			err = c.LifecycleConverter.ToHclLookupById(channel.LifecycleId, dependencies)
		}

		if err != nil {
			return err
		}
	}

	thisResource := ResourceDetails{}
	resourceName := "channel_" + sanitizer.SanitizeName(project.Name) + "_" + sanitizer.SanitizeNamePointer(&channel.Name)
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = channel.Id
	thisResource.ResourceType = c.GetResourceType()

	if channel.Name == "Default" {
		// TODO: Many channels are called default! But there is no way to look up a channel based on its project.
		thisResource.Lookup = "${data." + octopusdeployChannelDataType + "." + resourceName + ".channels[0].id}"
		thisResource.ToHcl = func() (string, error) {
			data := terraform.TerraformChannelData{
				Name:        resourceName,
				Type:        octopusdeployChannelDataType,
				Ids:         nil,
				PartialName: channel.Name,
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(data, "data")
			// Channel lookup really needs to be project specific before we can validate like this
			//hcl.WriteLifecyclePostCondition(block, "Failed to resolve a channel called \""+channel.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.channels) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	} else {

		if stateless {
			// TODO: because we can not retrieve a project specific channel from a data block, there is no good way
			// to construct a lookup here if the project exists. That said, if the project exists, no other resource
			// that might look up a channel (like project variables) will be created either, so nothing will ever use
			// the lookup. So we just use an empty string for the lookup.
			thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + resourceName + ".projects) != 0 " +
				"? '' " +
				": " + octopusdeployChannelResourceType + "." + resourceName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployChannelResourceType + "." + resourceName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeployChannelResourceType + "." + resourceName + ".id}"
		}

		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform.TerraformChannel{
				Type:         octopusdeployChannelResourceType,
				Name:         resourceName,
				ResourceName: channel.Name,
				Description:  channel.Description,
				LifecycleId:  c.getLifecycleId(channel.LifecycleId, dependencies),
				ProjectId:    dependencies.GetResource("Projects", channel.ProjectId),
				IsDefault:    channel.IsDefault,
				Rule:         c.convertRules(channel.Rules),
				TenantTags:   c.Excluder.FilteredTenantTags(channel.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			}
			file := hclwrite.NewEmptyFile()

			if stateless {
				// when importing a stateless project, the channel is only created if the project does not exist
				c.writeData(file, project.Name, resourceName)
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployProjectsDataType + "." + resourceName + ".projects) != 0 ? 0 : 1}")
			}

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), channel.Name, octopusdeployChannelResourceType, resourceName))

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			if stateless {
				hcl.WriteLifecyclePreventDeleteAttribute(block)
			}

			/* Channels reference steps and packages by text without terraform understanding
			there is any relationship. In order for the channel to be created after the deployment process,
			we must make this dependency explicit. Otherwise, the channel may be created without the deployment
			process, and Octopus will reject the channel rules.*/
			manualDependencies := make([]string, 0)
			for t, r := range terraformDependencies {
				if t != "" && r != "" {
					dependency := dependencies.GetResourceDependency(t, r)
					dependency = hcl.RemoveId(hcl.RemoveInterpolation(dependency))
					manualDependencies = append(manualDependencies, dependency)
				}
			}
			hcl.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(manualDependencies[:], ",")+"]")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	}
	dependencies.AddResource(thisResource)
	return nil
}

func (c ChannelConverter) getLifecycleId(lifecycleId string, dependencies *ResourceDetailsCollection) *string {
	if lifecycleId == "" {
		return nil
	}

	lifecycleLookup := dependencies.GetResource("Lifecycles", lifecycleId)
	return &lifecycleLookup
}

func (c ChannelConverter) GetResourceType() string {
	return "Channels"
}

func (c ChannelConverter) GetGroupResourceType(projectId string) string {
	return "Projects/" + projectId + "/channels"
}

func (c ChannelConverter) convertRules(rules []octopus.Rule) []terraform.TerraformRule {
	terraformRules := make([]terraform.TerraformRule, 0)
	for _, v := range rules {
		terraformRules = append(terraformRules, terraform.TerraformRule{
			ActionPackage: c.convertActionPackages(v.ActionPackages),
			Tag:           v.Tag,
			VersionRange:  v.VersionRange,
		})
	}
	return terraformRules
}

func (c ChannelConverter) convertActionPackages(actionPackages []octopus.ActionPackage) []terraform.TerraformActionPackage {
	collection := make([]terraform.TerraformActionPackage, 0)
	for _, v := range actionPackages {
		collection = append(collection, terraform.TerraformActionPackage{
			DeploymentAction: v.DeploymentAction,
			PackageReference: v.PackageReference,
		})
	}
	return collection
}
