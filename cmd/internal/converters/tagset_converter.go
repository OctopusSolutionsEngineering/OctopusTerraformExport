package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const octopusdeployTagSetResourceType = "octopusdeploy_tag_set"
const octopusdeployTagResourceType = "octopusdeploy_tag"
const octopusdeployTagSetsData = "octopusdeploy_tag_sets"

type TagSetConverter struct {
	Client                     client.OctopusClient
	ExcludeTenantTags          args.StringSliceArgs
	ExcludeTenantTagSets       args.StringSliceArgs
	ExcludeTenantTagSetsRegex  args.StringSliceArgs
	ExcludeTenantTagSetsExcept args.StringSliceArgs
	ExcludeAllTenantTagSets    bool
	Excluder                   ExcludeByName
	ErrGroup                   *errgroup.Group
	LimitResourceCount         int
	GenerateImportScripts      bool
}

func (c *TagSetConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c *TagSetConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c *TagSetConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTenantTagSets {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.TagSet]{
		Client: c.Client,
	}

	done := make(chan struct{})
	defer close(done)

	channel := batchClient.GetAllResourcesBatch(done, c.GetResourceType())

	for resourceWrapper := range channel {
		if resourceWrapper.Err != nil {
			return resourceWrapper.Err
		}

		resource := resourceWrapper.Res
		if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllTenantTagSets, c.ExcludeTenantTagSets, c.ExcludeTenantTagSetsRegex, c.ExcludeTenantTagSetsExcept) {
			continue
		}

		zap.L().Info("Tagset: " + resource.Id)
		err := c.toHcl(resource, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *TagSetConverter) ToHclByResource(tagSet octopus.TagSet, dependencies *data.ResourceDetailsCollection) error {
	return c.toHcl(tagSet, false, dependencies)
}

func (c *TagSetConverter) GetResourceType() string {
	return "TagSets"
}

// toBashImport creates a bash script to import the resource
func (c *TagSetConverter) toBashImport(resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
	dependencies.AddResource(data.ResourceDetails{
		FileName: "space_population/import_" + resourceName + ".sh",
		ToHcl: func() (string, error) {
			return fmt.Sprintf(`#!/bin/bash

# This script is used to import an exiting resource into the Terraform state.
# It is useful when importing a Terraform module into an Octopus space that
# already has existing resources.

# Make the script executable with the command:
# chmod +x ./import_%s.sh

# Alternativly, run the script with bash directly:
# /bin/bash ./import_%s.sh <options>

# Run "terraform init" to download any required providers and to configure the
# backend configuration

# Then run the import script. Replace the API key, instance URL, and Space ID 
# in the example below with the values of the space that the Terraform module 
# will be imported into.

# ./import_%s.sh API-xxxxxxxxxxxx https://yourinstance.octopus.app Spaces-1234

if [[ $# -ne 3 ]]
then
	echo "Usage: ./import_%s.sh <API Key> <Octopus URL> <Space ID>"
    echo "Example: ./import_%s.sh API-xxxxxxxxxxxx https://yourinstance.octopus.app Spaces-1234"
	exit 1
fi

if ! command -v jq &> /dev/null
then
    echo "jq is required"
    exit 1
fi

if ! command -v curl &> /dev/null
then
    echo "curl is required"
    exit 1
fi

RESOURCE_NAME="%s"
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/TagSets" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No tag set found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing tag set ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, octopusdeployTagSetResourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c *TagSetConverter) toPowershellImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
	dependencies.AddResource(data.ResourceDetails{
		FileName: "space_population/import_" + resourceName + ".ps1",
		ToHcl: func() (string, error) {
			return fmt.Sprintf(`# This script is used to import an exiting resource into the Terraform state.
# It is useful when importing a Terraform module into an Octopus space that
# already has existing resources.

# Run "terraform init" to download any required providers and to configure the
# backend configuration

# Then run the import script. Replace the API key, instance URL, and Space ID 
# in the example below with the values of the space that the Terraform module 
# will be imported into.

# ./import_%s.ps1 API-xxxxxxxxxxxx https://yourinstance.octopus.app Spaces-1234

param (
    [Parameter(Mandatory=$true)]
    [string]$ApiKey,

    [Parameter(Mandatory=$true)]
    [string]$Url,

    [Parameter(Mandatory=$true)]
    [string]$SpaceId
)

$ResourceName="%s"

$headers = @{
    "X-Octopus-ApiKey" = $ApiKey
}

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/TagSets?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No tag set found with the name $ResourceName"
	exit 1
}

echo "Importing tag set $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, octopusdeployTagSetResourceType, resourceName), nil
		},
	})
}

func (c *TagSetConverter) toHcl(tagSet octopus.TagSet, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.Excluder.IsResourceExcludedWithRegex(tagSet.Name, c.ExcludeAllTenantTagSets, c.ExcludeTenantTagSets, c.ExcludeTenantTagSetsRegex, c.ExcludeTenantTagSetsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + tagSet.Id)
		return nil
	}

	tagSetName := "tagset_" + sanitizer.SanitizeName(tagSet.Name)

	if c.GenerateImportScripts {
		c.toBashImport(tagSetName, tagSet.Name, dependencies)
		c.toPowershellImport(tagSetName, tagSet.Name, dependencies)
	}

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + tagSetName + ".tf"
	thisResource.Id = tagSet.Id
	thisResource.Name = tagSet.Name
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

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
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

		tagsetName := "tagset_" + sanitizer.SanitizeName(tagSet.Name)
		tagName := "tag_" + sanitizer.SanitizeName(tag.Name)

		tagResource := data.ResourceDetails{}
		tagResource.FileName = "space_population/" + tagsetName + "_" + tagName + ".tf"
		tagResource.Id = tag.Id
		tagResource.Name = tag.Name
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

func (c *TagSetConverter) buildData(resourceName string, resource octopus.TagSet) terraform.TerraformTagSetData {
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
func (c *TagSetConverter) writeData(file *hclwrite.File, resource octopus.TagSet, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}
