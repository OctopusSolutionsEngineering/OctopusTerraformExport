package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"strings"
)

// TenantTagDependencyGenerator deals with the unusual case where Octopus resources link to each other by name rather
// than by ID. This forces us to manually define the dependencies between resources, otherwise Terraform may create
// a resource like an account before it creates the tagset that the account references.
type TenantTagDependencyGenerator struct {
}

// AddAndWriteTagSetDependencies writes a depends_on block to a terraform resource, and optionally recursively includes
// the tagsets that the resource depends on.
func (t TenantTagDependencyGenerator) AddAndWriteTagSetDependencies(client client.OctopusClient, tenantTags []string, tagSetConverter TagSetConverter, block *hclwrite.Block, dependencies *data.ResourceDetailsCollection, recursive bool) error {
	collection := octopus.GeneralCollection[octopus.TagSet]{}
	err := client.GetAllResources("TagSets", &collection)

	if err != nil {
		return err
	}

	tagSets, tags, err := t.FindDependencies(tenantTags, collection)

	if err != nil {
		return err
	}

	err = t.WriteTagSetDependencies(tagSets, tags, block, dependencies)

	if err != nil {
		return err
	}

	if recursive {

		err = t.AddTagSetDependencies(tagSets, tagSetConverter, dependencies)

		return err
	}

	return nil
}

func (t TenantTagDependencyGenerator) AddTagSetDependencies(tagSets []octopus.TagSet, tagSetConverter TagSetConverter, dependencies *data.ResourceDetailsCollection) error {
	for _, tagSet := range tagSets {
		err := tagSetConverter.ToHclByResource(tagSet, dependencies)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t TenantTagDependencyGenerator) WriteTagSetDependencies(tagSets []octopus.TagSet, tags []octopus.Tag, block *hclwrite.Block, dependencies *data.ResourceDetailsCollection) error {
	// Explicitly describe the dependency between a variable and a tag set
	tagSetDependencies := []string{}
	if tagSets != nil {
		tagSetDependencies = lo.FilterMap(tagSets, func(item octopus.TagSet, index int) (string, bool) {
			dependency := dependencies.GetResourceDependency("TagSets", item.Id)
			return dependency, dependency != ""
		})
	}

	tagDependencies := []string{}
	if tags != nil {
		tagDependencies = lo.FilterMap(tags, func(item octopus.Tag, index int) (string, bool) {
			dependency := dependencies.GetResourceDependency("Tags", item.Id)
			return dependency, dependency != ""
		})
	}

	allDependencies := lo.Map(append(tagSetDependencies, tagDependencies...), func(item string, index int) string {
		return hcl.RemoveId(hcl.RemoveInterpolation(item))
	})

	hcl.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(allDependencies, ",")+"]")

	return nil
}

// FindDependencies returns the tag sets and tags that are references by the tenant tags
func (t TenantTagDependencyGenerator) FindDependencies(tenantTags []string, collection octopus.GeneralCollection[octopus.TagSet]) ([]octopus.TagSet, []octopus.Tag, error) {

	tagSets := []octopus.TagSet{}
	tags := []octopus.Tag{}

	if tenantTags == nil {
		return tagSets, tags, nil
	}

	for _, tagSet := range collection.Items {
		for _, tag := range tagSet.Tags {
			for _, tenantTag := range tenantTags {
				if tag.CanonicalTagName == tenantTag {
					if !lo.SomeBy(tagSets, func(item octopus.TagSet) bool {
						return item.Id == tagSet.Id
					}) {
						tagSets = append(tagSets, tagSet)
					}

					if !lo.SomeBy(tags, func(item octopus.Tag) bool {
						return item.Id == tagSet.Id
					}) {
						tags = append(tags, tag)
					}

				}
			}
		}
	}

	return tagSets, tags, nil
}
