package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
	"strings"
)

type ChannelConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	ProjectId         string
	ProjectLookup     string
	LifecycleMap      map[string]string
	DependsOn         []string
}

func (c ChannelConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.Channel]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	channelMap := map[string]string{}

	for _, channel := range collection.Items {
		resourceName := "channel_" + util.SanitizeNamePointer(&channel.Name)

		// Assume the default lifecycle already exists
		if channel.Name != "Default" {
			terraformResource := terraform.TerraformChannel{
				Type:         "octopusdeploy_channel",
				Name:         resourceName,
				ResourceName: channel.Name,
				Description:  channel.Description,
				LifecycleId:  c.getLifecycleId(channel.LifecycleId),
				ProjectId:    c.ProjectLookup,
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
			util.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(c.DependsOn[:], ",")+"]")
			file.Body().AppendBlock(block)

			results["space_population/"+resourceName+".tf"] = string(file.Bytes())
			channelMap[channel.Id] = "${octopusdeploy_channel." + resourceName + ".id}"
		}
	}

	return results, channelMap, nil
}

func (c ChannelConverter) getLifecycleId(lifecycleId string) *string {
	if lifecycleId == "" {
		return nil
	}

	lifecycleLookup := c.LifecycleMap[lifecycleId]
	return &lifecycleLookup
}

func (c ChannelConverter) GetResourceType() string {
	return "Projects/" + c.ProjectId + "/channels"
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
