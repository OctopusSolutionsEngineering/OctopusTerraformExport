package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
	"strings"
)

type ChannelConverter struct {
	Client             client.OctopusClient
	LifecycleConverter ConverterById
}

func (c ChannelConverter) ToHclByProjectIdWithTerraDependencies(projectId string, terraformDependencies map[string]string, dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.Channel]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return err
	}

	for _, channel := range collection.Items {
		err = c.toHcl(channel, true, terraformDependencies, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c ChannelConverter) toHcl(channel octopus2.Channel, recursive bool, terraformDependencies map[string]string, dependencies *ResourceDetailsCollection) error {
	if recursive && channel.LifecycleId != "" {
		// The lifecycle is a dependency that we need to lookup
		err := c.LifecycleConverter.ToHclById(channel.LifecycleId, dependencies)

		if err != nil {
			return err
		}
	}

	thisResource := ResourceDetails{}
	resourceName := "channel_" + sanitizer.SanitizeNamePointer(&channel.Name)
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = channel.Id
	thisResource.ResourceType = c.GetResourceType()

	if channel.Name == "Default" {
		// TODO: Many channels are called default! But there is no way to look up a channel based on its project.
		thisResource.Lookup = "${data.octopusdeploy_channels." + resourceName + ".channels[0].id}"
		thisResource.ToHcl = func() (string, error) {
			data := terraform2.TerraformChannelData{
				Name:        resourceName,
				Type:        "octopusdeploy_channels",
				Ids:         nil,
				PartialName: channel.Name,
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(data, "data"))

			return string(file.Bytes()), nil
		}
	} else {
		thisResource.Lookup = "${octopusdeploy_channel." + resourceName + ".id}"
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform2.TerraformChannel{
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

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: API-REPLACEME\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.name=\"" + channel.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_channel." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			/* Channels reference steps and packages by text without terraform understanding
			there is any relationship. In order for the channel to be created after the deployment process,
			we must make this dependency explicit. Otherwise, the channel may be created without the deployment
			process, and Octopus will reject the channel rules.*/
			manualDependencies := make([]string, 0)
			for t, r := range terraformDependencies {
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

func (c ChannelConverter) convertRules(rules []octopus2.Rule) []terraform2.TerraformRule {
	terraformRules := make([]terraform2.TerraformRule, 0)
	for _, v := range rules {
		terraformRules = append(terraformRules, terraform2.TerraformRule{
			ActionPackage: c.convertActionPackages(v.ActionPackages),
			Tag:           v.Tag,
			VersionRange:  v.VersionRange,
		})
	}
	return terraformRules
}

func (c ChannelConverter) convertActionPackages(actionPackages []octopus2.ActionPackage) []terraform2.TerraformActionPackage {
	collection := make([]terraform2.TerraformActionPackage, 0)
	for _, v := range actionPackages {
		collection = append(collection, terraform2.TerraformActionPackage{
			DeploymentAction: v.DeploymentAction,
			PackageReference: v.PackageReference,
		})
	}
	return collection
}
