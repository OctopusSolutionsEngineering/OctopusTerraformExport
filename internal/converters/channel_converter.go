package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/hcl"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/sanitizer"
	"strings"
)

type ChannelConverter struct {
	Client             client.OctopusClient
	DependsOn          map[string]string
	LifecycleConverter ConverterById
}

func (c ChannelConverter) ToHclByProjectId(projectId string, recursive bool, dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Channel]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return err
	}

	for _, channel := range collection.Items {
		err = c.toHcl(channel, recursive, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c ChannelConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	channel := octopus.Channel{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &channel)

	if err != nil {
		return err
	}

	return c.toHcl(channel, true, dependencies)
}

func (c ChannelConverter) toHcl(channel octopus.Channel, recursive bool, dependencies *ResourceDetailsCollection) error {
	if channel.Name != "Default" {

		if recursive {
			// The lifecycle is a dependency that we need to lookup
			err := c.LifecycleConverter.ToHclById(channel.LifecycleId, dependencies)

			if err != nil {
				return err
			}
		}

		resourceName := "channel_" + sanitizer.SanitizeNamePointer(&channel.Name)

		thisResource := ResourceDetails{}
		thisResource.FileName = "space_population/" + resourceName + ".tf"
		thisResource.Id = channel.Id
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = "${octopusdeploy_channel." + resourceName + ".id}"
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform.TerraformChannel{
				Type:         "octopusdeploy_channel",
				Name:         resourceName,
				ResourceName: channel.Name,
				Description:  channel.Description,
				LifecycleId:  c.getLifecycleId(channel.LifecycleId, dependencies),
				ProjectId:    dependencies.GetResource("Projects", channel.ProjectId),
				IsDefault:    channel.IsDefault,
				Rule:         c.convertRules(channel.Rules),
				TenantTags:   channel.TenantTags,
			}
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			/* Channels reference steps and packages by text without terraform understanding
			there is any relationship. In order for the channel to be created after the deployment process,
			we must make this dependency explicit. Otherwise, the channel may be created without the deployment
			process, and Octopus will reject the channel rules.*/
			manualDependencies := make([]string, 0)
			for t, r := range c.DependsOn {
				if t != "" && r != "" {
					dependency := dependencies.GetResource(t, r)
					// This is a raw expression, so remove the surrounding brackets
					dependency = strings.Replace(dependency, "${", "", -1)
					dependency = strings.Replace(dependency, ".id}", "", -1)
					manualDependencies = append(manualDependencies, dependency)
				}
			}
			hcl.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(manualDependencies[:], ",")+"]")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		dependencies.AddResource(thisResource)
	}

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
