package converters

import (
	"errors"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type FeedConverter struct {
	Client client.OctopusClient
}

func (c FeedConverter) GetResourceType() string {
	return "Feeds"
}

func (c FeedConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Feed]{}
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
	resource := octopus.Feed{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, true, dependencies)
}

func (c FeedConverter) toHcl(resource octopus.Feed, recursive bool, dependencies *ResourceDetailsCollection) error {
	resourceName := "feed_" + util.SanitizeName(resource.Name)
	passwordName := resourceName + "_password"
	password := "${var." + passwordName + "}"

	thisResource := ResourceDetails{}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	if util.EmptyIfNil(resource.FeedType) == "BuiltIn" {
		thisResource.Lookup = "${data.octopusdeploy_feeds.built_in_feed.feeds[0].id}"
	} else if util.EmptyIfNil(resource.FeedType) == "Docker" {
		thisResource.Lookup = "${octopusdeploy_docker_container_registry." + resourceName + ".id}"
	} else if util.EmptyIfNil(resource.FeedType) == "AwsElasticContainerRegistry" {
		thisResource.Lookup = "${octopusdeploy_aws_elastic_container_registry." + resourceName + ".id}"
	} else if util.EmptyIfNil(resource.FeedType) == "Maven" {
		thisResource.Lookup = "${octopusdeploy_maven_feed." + resourceName + ".id}"
	} else if util.EmptyIfNil(resource.FeedType) == "GitHub" {
		thisResource.Lookup = "${octopusdeploy_github_repository_feed." + resourceName + ".id}"
	} else if util.EmptyIfNil(resource.FeedType) == "Helm" {
		thisResource.Lookup = "${octopusdeploy_helm_feed." + resourceName + ".id}"
	} else if util.EmptyIfNil(resource.FeedType) == "NuGet" {
		thisResource.Lookup = "${octopusdeploy_nuget_feed." + resourceName + ".id}"
	}
	thisResource.ToHcl = func() (string, error) {

		if util.EmptyIfNil(resource.FeedType) == "BuiltIn" {
			terraformResource := terraform.TerraformFeedData{
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

		if util.EmptyIfNil(resource.FeedType) == "Docker" {
			terraformResource := terraform.TerraformDockerFeed{
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

			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if util.EmptyIfNil(resource.FeedType) == "AwsElasticContainerRegistry" {
			terraformResource := terraform.TerraformEcrFeed{
				Type:                              "octopusdeploy_aws_elastic_container_registry",
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				AccessKey:                         resource.AccessKey,
				SecretKey:                         &password,
				Region:                            resource.Region,
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if util.EmptyIfNil(resource.FeedType) == "Maven" {
			terraformResource := terraform.TerraformMavenFeed{
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

			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if util.EmptyIfNil(resource.FeedType) == "GitHub" {
			terraformResource := terraform.TerraformGitHubRepoFeed{
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

			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if util.EmptyIfNil(resource.FeedType) == "Helm" {
			terraformResource := terraform.TerraformHelmFeed{
				Type:                              "octopusdeploy_helm_feed",
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				FeedUri:                           resource.FeedUri,
				Username:                          resource.Username,
				Password:                          &password,
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if util.EmptyIfNil(resource.FeedType) == "NuGet" {
			terraformResource := terraform.TerraformNuGetFeed{
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

			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if util.EmptyIfNil(resource.FeedType) == "OctopusProject" {
			// We don't do anything with this feed
			return "", nil
		}

		return "", errors.New("found unexpected feed type: " + util.EmptyIfNil(resource.FeedType))
	}

	dependencies.AddResource(thisResource)
	return nil
}
