package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/strutil"
)

type TagSetConverter struct {
	Client client.OctopusClient
}

func (c TagSetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.TagSet]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, tagSet := range collection.Items {
		err = c.ToHclByResource(tagSet, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c TagSetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	tagSet := octopus2.TagSet{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &tagSet)

	if err != nil {
		return err
	}

	return c.ToHclByResource(tagSet, dependencies)
}

func (c TagSetConverter) ToHclByResource(tagSet octopus2.TagSet, dependencies *ResourceDetailsCollection) error {
	tagSetName := "tagset_" + sanitizer.SanitizeName(tagSet.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + tagSetName + ".tf"
	thisResource.Id = tagSet.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_tag_set." + tagSetName + ".id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformTagSet{
			Type:         "octopusdeploy_tag_set",
			Name:         tagSetName,
			ResourceName: tagSet.Name,
			Description:  strutil.NilIfEmptyPointer(tagSet.Description),
			SortOrder:    tagSet.SortOrder,
		}
		file := hclwrite.NewEmptyFile()

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
			Type: hclsyntax.TokenComment,
			Bytes: []byte("# Import existing resources with the following commands:\n" +
				"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: API-REPLACEME\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.name=\"" + tagSet.Name + "\") | .Id')\n" +
				"# terraform import octopusdeploy_tag_set." + tagSetName + " ${RESOURCE_ID}\n"),
			SpacesBefore: 0,
		}})

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)

	for _, tag := range tagSet.Tags {
		// capture the tag for the function literal below.
		// https://go.dev/doc/faq#closures_and_goroutines
		tag := tag

		tagName := "tag_" + sanitizer.SanitizeName(tag.Name)

		tagResource := ResourceDetails{}
		tagResource.FileName = "space_population/" + tagName + ".tf"
		tagResource.Id = tag.Id
		tagResource.ResourceType = "Tags"
		tagResource.Lookup = "${octopusdeploy_tag." + tagName + ".id}"
		tagResource.ToHcl = func() (string, error) {
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

			return string(file.Bytes()), nil
		}
		dependencies.AddResource(tagResource)
	}

	return nil
}

func (c TagSetConverter) GetResourceType() string {
	return "TagSets"
}

func (c TagSetConverter) getTagSetTags(tags []octopus2.Tag) []terraform.TerraformTag {
	terraformTags := make([]terraform.TerraformTag, len(tags))
	for i, v := range tags {
		terraformTags[i] = terraform.TerraformTag{
			ResourceName: v.Name,
			Color:        v.Color,
			Description:  strutil.NilIfEmptyPointer(v.Description),
			SortOrder:    v.SortOrder,
		}
	}
	return terraformTags
}
