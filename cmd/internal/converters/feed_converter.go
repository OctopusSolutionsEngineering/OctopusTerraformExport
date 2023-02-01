package converters

import (
	"errors"
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/strutil"
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
		err = c.toHcl(resource, false, dependencies)

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

	return c.toHcl(resource, true, dependencies)
}

func (c FeedConverter) toHcl(resource octopus2.Feed, recursive bool, dependencies *ResourceDetailsCollection) error {
	resourceName := "feed_" + sanitizer.SanitizeName(resource.Name)
	passwordName := resourceName + "_password"
	password := "${var." + passwordName + "}"

	thisResource := ResourceDetails{}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	if strutil.EmptyIfNil(resource.FeedType) == "BuiltIn" {
		thisResource.Lookup = "${data.octopusdeploy_feeds.built_in_feed.feeds[0].id}"
	} else if strutil.EmptyIfNil(resource.FeedType) == "Docker" {
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

		if strutil.EmptyIfNil(resource.FeedType) == "BuiltIn" {
			terraformResource := terraform2.TerraformFeedData{
				Type:     "octopusdeploy_feeds",
				Name:     "built_in_feed",
				FeedType: "BuiltIn",
				Skip:     0,
				Take:     1,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

			return string(file.Bytes()), nil
		}

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
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: API-REPLACEME\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.name=\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_docker_container_registry." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if strutil.EmptyIfNil(resource.FeedType) == "AwsElasticContainerRegistry" {
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
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: API-REPLACEME\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.name=\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_aws_elastic_container_registry." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if strutil.EmptyIfNil(resource.FeedType) == "Maven" {
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
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: API-REPLACEME\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.name=\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_maven_feed." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if strutil.EmptyIfNil(resource.FeedType) == "GitHub" {
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
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: API-REPLACEME\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.name=\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_github_repository_feed." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if strutil.EmptyIfNil(resource.FeedType) == "Helm" {
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
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: API-REPLACEME\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.name=\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_helm_feed." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if strutil.EmptyIfNil(resource.FeedType) == "NuGet" {
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
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: API-REPLACEME\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.name=\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_nuget_feed." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if strutil.EmptyIfNil(resource.FeedType) == "OctopusProject" {
			// We don't do anything with this feed
			return "", nil
		}

		fmt.Println(errors.New("Found unexpected feed type \"" + strutil.EmptyIfNil(resource.FeedType) + "\" with name \"" + resource.Name + "\"."))

		return "", nil
	}

	dependencies.AddResource(thisResource)
	return nil
}
