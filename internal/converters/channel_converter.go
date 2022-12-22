package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type ChannelConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	ProjectId         string
	LifecycleMap      map[string]string
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
		resourceName := "channel_" + util.SanitizeName(channel.Slug)

		// Assume the default lifecycle already exists
		if channel.Name != "Default" {
			terraformResource := terraform.TerraformChannel{
				Type:         "octopusdeploy_channel",
				Name:         resourceName,
				SpaceId:      c.SpaceResourceName,
				ResourceName: channel.Name,
				Description:  channel.Description,
				LifecycleId:  c.LifecycleMap[channel.LifecycleId],
				ProjectId:    c.ProjectId,
				IsDefault:    channel.IsDefault,
				Rule:         c.convertRules(channel.Rules),
				TenantTags:   channel.TenantTags,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			results[resourceName+".tf"] = string(file.Bytes())
			channelMap[channel.Id] = "${octopusdeploy_channel." + resourceName + ".id}"
		}
	}

	return results, channelMap, nil
}

func (c ChannelConverter) ToHclByName(name string) (map[string]string, error) {
	return map[string]string{}, nil
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
