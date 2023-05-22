package converters

import (
	"errors"
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
)

type FeedConverter struct {
	Client client.OctopusClient
}

func (c FeedConverter) GetResourceType() string {
	return "Feeds"
}

func (c FeedConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.Feed]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		err = c.toHcl(resource, false, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c FeedConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.Feed{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, true, false, dependencies)
}

func (c FeedConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.Feed{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, false, true, dependencies)
}

func (c FeedConverter) toHcl(resource octopus2.Feed, recursive bool, lookup bool, dependencies *ResourceDetailsCollection) error {
	forceLookup := lookup || strutil.EmptyIfNil(resource.FeedType) == "BuiltIn"

	resourceName := "feed_" + sanitizer.SanitizeName(resource.Name)

	thisResource := ResourceDetails{}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()

	if forceLookup {
		c.toHclLookup(resource, &thisResource, resourceName)
	} else {
		c.toHclResource(resource, &thisResource, resourceName)
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c FeedConverter) toHclResource(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) {
	passwordName := resourceName + "_password"
	password := "${var." + passwordName + "}"

	if strutil.EmptyIfNil(resource.FeedType) == "Docker" {
		thisResource.Lookup = "${octopusdeploy_docker_container_registry." + resourceName + ".id}"
	} else if strutil.EmptyIfNil(resource.FeedType) == "AwsElasticContainerRegistry" {
		thisResource.Lookup = "${octopusdeploy_aws_elastic_container_registry." + resourceName + ".id}"
	} else if strutil.EmptyIfNil(resource.FeedType) == "Maven" {
		thisResource.Lookup = "${octopusdeploy_maven_feed." + resourceName + ".id}"
	} else if strutil.EmptyIfNil(resource.FeedType) == "GitHub" {
		thisResource.Lookup = "${octopusdeploy_github_repository_feed." + resourceName + ".id}"
	} else if strutil.EmptyIfNil(resource.FeedType) == "Helm" {
		thisResource.Lookup = "${octopusdeploy_helm_feed." + resourceName + ".id}"
	} else if strutil.EmptyIfNil(resource.FeedType) == "NuGet" {
		thisResource.Lookup = "${octopusdeploy_nuget_feed." + resourceName + ".id}"
	}
	thisResource.ToHcl = func() (string, error) {

		if strutil.EmptyIfNil(resource.FeedType) == "Docker" {
			terraformResource := terraform2.TerraformDockerFeed{
				Type:                              "octopusdeploy_docker_container_registry",
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				Password:                          &password,
				RegistryPath:                      resource.RegistryPath,
				Username:                          resource.Username,
				ApiVersion:                        resource.ApiVersion,
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
				FeedUri:                           resource.FeedUri,
			}

			secretVariableResource := terraform2.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_docker_container_registry." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		} else if strutil.EmptyIfNil(resource.FeedType) == "AwsElasticContainerRegistry" {
			terraformResource := terraform2.TerraformEcrFeed{
				Type:                              "octopusdeploy_aws_elastic_container_registry",
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				AccessKey:                         resource.AccessKey,
				SecretKey:                         &password,
				Region:                            resource.Region,
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
			}

			secretVariableResource := terraform2.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_aws_elastic_container_registry." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		} else if strutil.EmptyIfNil(resource.FeedType) == "Maven" {
			terraformResource := terraform2.TerraformMavenFeed{
				Type:                              "octopusdeploy_maven_feed",
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				FeedUri:                           resource.FeedUri,
				Username:                          resource.Username,
				Password:                          &password,
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
				DownloadAttempts:                  resource.DownloadAttempts,
				DownloadRetryBackoffSeconds:       resource.DownloadRetryBackoffSeconds,
			}

			secretVariableResource := terraform2.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_maven_feed." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		} else if strutil.EmptyIfNil(resource.FeedType) == "GitHub" {
			terraformResource := terraform2.TerraformGitHubRepoFeed{
				Type:                              "octopusdeploy_github_repository_feed",
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				FeedUri:                           resource.FeedUri,
				Username:                          resource.Username,
				Password:                          &password,
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
				DownloadAttempts:                  resource.DownloadAttempts,
				DownloadRetryBackoffSeconds:       resource.DownloadRetryBackoffSeconds,
			}

			secretVariableResource := terraform2.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_github_repository_feed." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		} else if strutil.EmptyIfNil(resource.FeedType) == "Helm" {
			terraformResource := terraform2.TerraformHelmFeed{
				Type:                              "octopusdeploy_helm_feed",
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				FeedUri:                           resource.FeedUri,
				Username:                          resource.Username,
				Password:                          &password,
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
			}

			secretVariableResource := terraform2.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_helm_feed." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		} else if strutil.EmptyIfNil(resource.FeedType) == "NuGet" {
			terraformResource := terraform2.TerraformNuGetFeed{
				Type:                              "octopusdeploy_nuget_feed",
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				FeedUri:                           resource.FeedUri,
				Username:                          resource.Username,
				Password:                          &password,
				IsEnhancedMode:                    resource.EnhancedMode,
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
				DownloadAttempts:                  resource.DownloadAttempts,
				DownloadRetryBackoffSeconds:       resource.DownloadRetryBackoffSeconds,
			}

			secretVariableResource := terraform2.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_nuget_feed." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		} else if strutil.EmptyIfNil(resource.FeedType) == "OctopusProject" {
			// We don't do anything with this feed
			return "", nil
		} else {
			fmt.Println(errors.New("Found unexpected feed type \"" + strutil.EmptyIfNil(resource.FeedType) + "\" with name \"" + resource.Name + "\"."))
		}

		return "", nil
	}
}

func (c FeedConverter) toHclLookup(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) {
	thisResource.Lookup = "${data.octopusdeploy_feeds." + resourceName + ".feeds[0].id}"

	if strutil.EmptyIfNil(resource.FeedType) == "BuiltIn" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform2.TerraformFeedData{
				Type:     "octopusdeploy_feeds",
				Name:     resourceName,
				FeedType: "BuiltIn",
				Skip:     0,
				Take:     1,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

			return string(file.Bytes()), nil
		}
	} else if strutil.EmptyIfNil(resource.FeedType) == "Docker" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform2.TerraformFeedData{
				Type:        "octopusdeploy_feeds",
				Name:        resourceName,
				PartialName: resource.Name,
				FeedType:    "Docker",
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

			return string(file.Bytes()), nil
		}
	} else if strutil.EmptyIfNil(resource.FeedType) == "AwsElasticContainerRegistry" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform2.TerraformFeedData{
				Type:        "octopusdeploy_feeds",
				Name:        resourceName,
				PartialName: resource.Name,
				FeedType:    "AwsElasticContainerRegistry",
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

			return string(file.Bytes()), nil
		}
	} else if strutil.EmptyIfNil(resource.FeedType) == "Maven" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform2.TerraformFeedData{
				Type:        "octopusdeploy_feeds",
				Name:        resourceName,
				PartialName: resource.Name,
				FeedType:    "Maven",
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

			return string(file.Bytes()), nil
		}
	} else if strutil.EmptyIfNil(resource.FeedType) == "GitHub" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform2.TerraformFeedData{
				Type:        "octopusdeploy_feeds",
				Name:        resourceName,
				PartialName: resource.Name,
				FeedType:    "GitHub",
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

			return string(file.Bytes()), nil
		}
	} else if strutil.EmptyIfNil(resource.FeedType) == "Helm" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform2.TerraformFeedData{
				Type:        "octopusdeploy_feeds",
				Name:        resourceName,
				PartialName: resource.Name,
				FeedType:    "Helm",
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

			return string(file.Bytes()), nil
		}
	} else if strutil.EmptyIfNil(resource.FeedType) == "NuGet" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform2.TerraformFeedData{
				Type:        "octopusdeploy_feeds",
				Name:        resourceName,
				PartialName: resource.Name,
				FeedType:    "NuGet",
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

			return string(file.Bytes()), nil
		}
	} else if strutil.EmptyIfNil(resource.FeedType) == "OctopusProject" {
		// We don't do anything with this feed
	} else {
		fmt.Println(errors.New("Found unexpected feed type \"" + strutil.EmptyIfNil(resource.FeedType) + "\" with name \"" + resource.Name + "\"."))
	}
}
