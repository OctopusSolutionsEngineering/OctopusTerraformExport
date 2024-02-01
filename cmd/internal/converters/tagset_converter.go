package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

const octopusdeployTagSetResourceType = "octopusdeploy_tag_set"
const octopusdeployTagResourceType = "octopusdeploy_tag"

type TagSetConverter struct {
	Client               client.OctopusClient
	ExcludeTenantTags    args.ExcludeTenantTags
	ExcludeTenantTagSets args.ExcludeTenantTagSets
	Excluder             ExcludeByName
}

func (c TagSetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.TagSet]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Tagset: " + resource.Id)
		err = c.ToHclByResource(resource, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c TagSetConverter) ToHclByResource(tagSet octopus2.TagSet, dependencies *ResourceDetailsCollection) error {

	if c.Excluder.IsResourceExcluded(tagSet.Name, false, c.ExcludeTenantTagSets, nil) {
		return nil
	}

	tagSetName := "tagset_" + sanitizer.SanitizeName(tagSet.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + tagSetName + ".tf"
	thisResource.Id = tagSet.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${" + octopusdeployTagSetResourceType + "." + tagSetName + ".id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformTagSet{
			Type:         octopusdeployTagSetResourceType,
			Name:         tagSetName,
			ResourceName: tagSet.Name,
			Description:  strutil.NilIfEmptyPointer(tagSet.Description),
			SortOrder:    tagSet.SortOrder,
		}
		file := hclwrite.NewEmptyFile()

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), tagSet.Name, octopusdeployTagSetResourceType, tagSetName))

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)

	for _, tag := range tagSet.Tags {
		if c.Excluder.IsResourceExcluded(tagSet.Name+"/"+tag.Name, false, c.ExcludeTenantTags, nil) {
			continue
		}

		// capture the tag for the function literal below.
		// https://go.dev/doc/faq#closures_and_goroutines
		tag := tag

		tagName := "tag_" + sanitizer.SanitizeName(tag.Name)

		tagResource := ResourceDetails{}
		tagResource.FileName = "space_population/" + tagName + ".tf"
		tagResource.Id = tag.Id
		tagResource.ResourceType = "Tags"
		tagResource.Lookup = "${" + octopusdeployTagResourceType + "." + tagName + ".id}"
		tagResource.ToHcl = func() (string, error) {
			terraformResource := terraform.TerraformTag{
				Type:         octopusdeployTagResourceType,
				Name:         tagName,
				ResourceName: tag.Name,
				TagSetId:     "${" + octopusdeployTagSetResourceType + "." + tagSetName + ".id}",
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
