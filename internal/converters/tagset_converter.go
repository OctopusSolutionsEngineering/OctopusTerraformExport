package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type TagSetConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
}

func (c TagSetConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.TagSet]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}

	for _, tagSet := range collection.Items {
		tagSetName := "tagset_" + util.SanitizeName(tagSet.Name)

		terraformResource := terraform.TerraformTagSet{
			Type:         "octopusdeploy_tag_set",
			Name:         tagSetName,
			ResourceName: tagSet.Name,
			Description:  util.NilIfEmptyPointer(tagSet.Description),
			SortOrder:    tagSet.SortOrder,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		results["space_population/tagset_"+tagSetName+".tf"] = string(file.Bytes())
		resultsMap[tagSet.Id] = "${octopusdeploy_tag_set." + tagSetName + ".id}"

		for _, tag := range tagSet.Tags {
			tagName := "tag_" + util.SanitizeName(tag.Name)
			terraformResource := terraform.TerraformTag{
				Type:         "octopusdeploy_tag",
				Name:         tagName,
				ResourceName: tag.Name,
				TagSetId:     "${octopusdeploy_tag_set." + tagSetName + ".id}",
				Color:        tag.Color,
				Description:  tag.Description,
				SortOrder:    tag.SortOrder,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			results["space_population/tag_"+tagName+".tf"] = string(file.Bytes())
			resultsMap[tag.Id] = "${octopusdeploy_tag." + tagName + ".id}"
		}
	}

	return results, resultsMap, nil
}

func (c TagSetConverter) GetResourceType() string {
	return "TagSets"
}

func (c TagSetConverter) getTagSetTags(tags []octopus.Tag) []terraform.TerraformTag {
	terraformTags := make([]terraform.TerraformTag, len(tags))
	for i, v := range tags {
		terraformTags[i] = terraform.TerraformTag{
			ResourceName: v.Name,
			Color:        v.Color,
			Description:  util.NilIfEmptyPointer(v.Description),
			SortOrder:    v.SortOrder,
		}
	}
	return terraformTags
}
