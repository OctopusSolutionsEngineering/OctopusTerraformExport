package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type FeedConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
}

func (c FeedConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.Feed]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}

	for _, resource := range collection.Items {
		feed, feedIdMap := c.convertFeed(resource)

		// merge the maps
		for k, v := range feed {
			results[k] = v
		}

		// merge the maps
		for k, v := range feedIdMap {
			resultsMap[k] = v
		}
	}

	return results, resultsMap, nil
}

func (c FeedConverter) ToHclById(id string) (map[string]string, map[string]string, error) {
	resource := octopus.Feed{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil, nil, err
	}

	feed, feedIdMap := c.convertFeed(resource)

	return feed, feedIdMap, nil
}

func (c FeedConverter) ToHclByName(name string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (c FeedConverter) GetResourceType() string {
	return "Feeds"
}

func (c FeedConverter) convertFeed(resource octopus.Feed) (map[string]string, map[string]string) {
	resourceName := "feed_" + util.SanitizeName(resource.Slug)
	password := "${var." + resourceName + "_password}"

	if *resource.FeedType == "BuiltIn" {
		terraformResource := terraform.TerraformFeedData{
			Type:     "octopusdeploy_feeds",
			Name:     "built_in_feed",
			FeedType: "BuiltIn",
			Skip:     0,
			Take:     1,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return map[string]string{
			resourceName + ".tf": string(file.Bytes()),
		}, map[string]string{resource.Id: "${data.octopusdeploy_feeds.feeds[0].id}"}
	}

	if *resource.FeedType == "Docker" {
		terraformResource := terraform.TerraformDockerFeed{
			Type:                              "octopusdeploy_docker_container_registry",
			Name:                              resourceName,
			ResourceName:                      resource.Name,
			Password:                          &password,
			RegistryPath:                      resource.RegistryPath,
			Username:                          resource.Username,
			ApiVersion:                        resource.ApiVersion,
			SpaceId:                           "${octopusdeploy_space." + c.SpaceResourceName + ".id}",
			PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return map[string]string{
			resourceName + ".tf": string(file.Bytes()),
		}, map[string]string{resource.Id: "${octopusdeploy_docker_container_registry." + resourceName + ".id}"}
	}

	if *resource.FeedType == "AwsElasticContainerRegistry" {
		terraformResource := terraform.TerraformEcrFeed{
			Type:                              "octopusdeploy_aws_elastic_container_registry",
			Name:                              resourceName,
			ResourceName:                      resource.Name,
			AccessKey:                         resource.AccessKey,
			SecretKey:                         &password,
			Region:                            nil,
			SpaceId:                           "${octopusdeploy_space." + c.SpaceResourceName + ".id}",
			PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return map[string]string{
			resourceName + ".tf": string(file.Bytes()),
		}, map[string]string{resource.Id: "${octopusdeploy_aws_elastic_container_registry." + resourceName + ".id}"}
	}

	if *resource.FeedType == "Maven" {
		terraformResource := terraform.TerraformMavenFeed{
			Type:                              "octopusdeploy_maven_feed",
			Name:                              resourceName,
			ResourceName:                      resource.Name,
			FeedUri:                           resource.FeedUri,
			SpaceId:                           "${octopusdeploy_space." + c.SpaceResourceName + ".id}",
			Username:                          resource.Username,
			Password:                          &password,
			PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
			DownloadAttempts:                  resource.DownloadAttempts,
			DownloadRetryBackoffSeconds:       resource.DownloadRetryBackoffSeconds,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return map[string]string{
			resourceName + ".tf": string(file.Bytes()),
		}, map[string]string{resource.Id: "${octopusdeploy_maven_feed." + resourceName + ".id}"}
	}

	if *resource.FeedType == "GitHub" {
		terraformResource := terraform.TerraformGitHubRepoFeed{
			Type:                              "octopusdeploy_github_repository_feed",
			Name:                              resourceName,
			ResourceName:                      resource.Name,
			FeedUri:                           resource.FeedUri,
			SpaceId:                           "${octopusdeploy_space." + c.SpaceResourceName + ".id}",
			Username:                          resource.Username,
			Password:                          &password,
			PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
			DownloadAttempts:                  resource.DownloadAttempts,
			DownloadRetryBackoffSeconds:       resource.DownloadRetryBackoffSeconds,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return map[string]string{
			resourceName + ".tf": string(file.Bytes()),
		}, map[string]string{resource.Id: "${octopusdeploy_github_repository_feed." + resourceName + ".id}"}
	}

	if *resource.FeedType == "Helm" {
		terraformResource := terraform.TerraformHelmFeed{
			Type:                              "octopusdeploy_helm_feed",
			Name:                              resourceName,
			ResourceName:                      resource.Name,
			FeedUri:                           resource.FeedUri,
			SpaceId:                           "${octopusdeploy_space." + c.SpaceResourceName + ".id}",
			Username:                          resource.Username,
			Password:                          &password,
			PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return map[string]string{
			resourceName + ".tf": string(file.Bytes()),
		}, map[string]string{resource.Id: "${octopusdeploy_helm_feed." + resourceName + ".id}"}
	}

	if *resource.FeedType == "NuGet" {
		terraformResource := terraform.TerraformNuGetFeed{
			Type:                              "octopusdeploy_nuget_feed",
			Name:                              resourceName,
			ResourceName:                      resource.Name,
			FeedUri:                           resource.FeedUri,
			SpaceId:                           "${octopusdeploy_space." + c.SpaceResourceName + ".id}",
			Username:                          resource.Username,
			Password:                          &password,
			IsEnhancedMode:                    resource.EnhancedMode,
			PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
			DownloadAttempts:                  resource.DownloadAttempts,
			DownloadRetryBackoffSeconds:       resource.DownloadRetryBackoffSeconds,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return map[string]string{
			resourceName + ".tf": string(file.Bytes()),
		}, map[string]string{resource.Id: "${octopusdeploy_nuget_feed." + resourceName + ".id}"}
	}

	return map[string]string{}, map[string]string{}
}
