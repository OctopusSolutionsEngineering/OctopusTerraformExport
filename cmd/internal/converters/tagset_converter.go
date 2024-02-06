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
	"go.uber.org/zap"
)

const octopusdeployTagSetResourceType = "octopusdeploy_tag_set"
const octopusdeployTagResourceType = "octopusdeploy_tag"
const octopusdeployTagSetsData = "octopusdeploy_tag_sets"

type TagSetConverter struct {
	Client               client.OctopusClient
	ExcludeTenantTags    args.ExcludeTenantTags
	ExcludeTenantTagSets args.ExcludeTenantTagSets
	Excluder             ExcludeByName
}

func (c *TagSetConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) error {
	return c.allToHcl(false, dependencies)
}

func (c *TagSetConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) error {
	return c.allToHcl(true, dependencies)
}

func (c *TagSetConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.TagSet]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Tagset: " + resource.Id)
		err = c.toHcl(resource, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *TagSetConverter) ToHclByResource(tagSet octopus2.TagSet, dependencies *data.ResourceDetailsCollection) error {
	return c.toHcl(tagSet, false, dependencies)
}

func (c *TagSetConverter) GetResourceType() string {
	return "TagSets"
}

func (c *TagSetConverter) toHcl(tagSet octopus2.TagSet, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.Excluder.IsResourceExcluded(tagSet.Name, false, c.ExcludeTenantTagSets, nil) {
		return nil
	}

	tagSetName := "tagset_" + sanitizer.SanitizeName(tagSet.Name)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + tagSetName + ".tf"
	thisResource.Id = tagSet.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${" + octopusdeployTagSetResourceType + "." + tagSetName + ".id}"

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployTagSetsData + "." + tagSetName + ".tag_sets) != 0 " +
			"? data." + octopusdeployTagSetsData + "." + tagSetName + ".tag_sets[0].id " +
			": " + octopusdeployTagSetResourceType + "." + tagSetName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployTagSetResourceType + "." + tagSetName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployTagSetResourceType + "." + tagSetName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformTagSet{
			Type:         octopusdeployTagSetResourceType,
			Name:         tagSetName,
			Count:        c.getCount(stateless, tagSetName),
			ResourceName: tagSet.Name,
			Description:  strutil.NilIfEmptyPointer(tagSet.Description),
			SortOrder:    tagSet.SortOrder,
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, tagSet, tagSetName)
		}

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), tagSet.Name, octopusdeployTagSetResourceType, tagSetName))

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDeleteAttribute(block)
		}

		file.Body().AppendBlock(block)

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

		tagResource := data.ResourceDetails{}
		tagResource.FileName = "space_population/" + tagName + ".tf"
		tagResource.Id = tag.Id
		tagResource.ResourceType = "Tags"
		tagResource.Lookup = c.getLookup(stateless, tagSetName, tagName)
		tagResource.Dependency = c.getDependency(stateless, tagName)

		tagResource.ToHcl = func() (string, error) {
			terraformResource := terraform.TerraformTag{
				Type:         octopusdeployTagResourceType,
				Name:         tagName,
				Count:        c.getCount(stateless, tagSetName),
				ResourceName: tag.Name,
				TagSetId:     c.getTagsetId(stateless, tagSetName, tagName),
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

func (c *TagSetConverter) getTagsetId(stateless bool, tagSetName string, tagName string) string {
	if stateless {
		return "${length(data." + octopusdeployTagSetsData + "." + tagSetName + ".tag_sets) != 0 " +
			"? data." + octopusdeployTagSetsData + "." + tagSetName + ".tag_sets[0].id " +
			": " + octopusdeployTagSetResourceType + "." + tagSetName + "[0].id}"
	}
	return "${" + octopusdeployTagSetResourceType + "." + tagSetName + ".id}"
}

func (c *TagSetConverter) getLookup(stateless bool, tagSetName string, tagName string) string {
	if stateless {
		// There is no tag lookup, so if the tagset exists, the tag is not created, and the lookup is an
		// empty string.
		return "${length(data." + octopusdeployTagSetsData + "." + tagSetName + ".tag_sets) != 0 " +
			"? '' " +
			": " + octopusdeployTagResourceType + "." + tagName + "[0].id}"
	}
	return "${" + octopusdeployTagResourceType + "." + tagName + ".id}"
}

func (c *TagSetConverter) getDependency(stateless bool, tagName string) string {
	if stateless {
		return "${" + octopusdeployTagResourceType + "." + tagName + "}"
	}
	return ""
}

func (c *TagSetConverter) getCount(stateless bool, tagSetName string) *string {
	if stateless {
		return strutil.StrPointer("${length(data." + octopusdeployTagSetsData + "." + tagSetName + ".tag_sets) != 0 ? 0 : 1}")
	}

	return nil
}

func (c *TagSetConverter) buildData(resourceName string, resource octopus2.TagSet) terraform.TerraformTagSetData {
	return terraform.TerraformTagSetData{
		Type:        octopusdeployTagSetsData,
		Name:        resourceName,
		Ids:         nil,
		PartialName: resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c *TagSetConverter) writeData(file *hclwrite.File, resource octopus2.TagSet, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}
