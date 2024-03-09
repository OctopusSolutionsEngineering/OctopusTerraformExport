package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const octopusdeployFeedsDataType = "octopusdeploy_feeds"
const octopusdeployDockerContainerRegistryResourceType = "octopusdeploy_docker_container_registry"
const octopusdeployAwsElasticContainerRegistryResourceType = "octopusdeploy_aws_elastic_container_registry"
const octopusdeployMavenFeedResourceType = "octopusdeploy_maven_feed"
const octopusdeployGithubRepositoryFeedResourceType = "octopusdeploy_github_repository_feed"
const octopusdeployHelmFeedResourceType = "octopusdeploy_helm_feed"
const octopusdeploy_nuget_feed_resource_type = "octopusdeploy_nuget_feed"

type FeedConverter struct {
	Client                    client.OctopusClient
	DummySecretVariableValues bool
	DummySecretGenerator      DummySecretGenerator
	ErrGroup                  *errgroup.Group
	ExcludeFeeds              args.StringSliceArgs
	ExcludeFeedsRegex         args.StringSliceArgs
	ExcludeFeedsExcept        args.StringSliceArgs
	ExcludeAllFeeds           bool
	Excluder                  ExcludeByName
}

func (c FeedConverter) GetResourceType() string {
	return "Feeds"
}

func (c FeedConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c FeedConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c FeedConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.Feed]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllFeeds, c.ExcludeFeeds, c.ExcludeFeedsRegex, c.ExcludeFeedsExcept) {
			continue
		}

		zap.L().Info("Feed: " + resource.Id)
		err = c.toHcl(resource, false, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c FeedConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c FeedConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c FeedConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
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

	zap.L().Info("Feed: " + resource.Id)
	return c.toHcl(resource, true, false, stateless, dependencies)
}

func (c FeedConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
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

	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllFeeds, c.ExcludeFeeds, c.ExcludeFeedsRegex, c.ExcludeFeedsExcept) {
		return nil
	}

	return c.toHcl(resource, false, true, false, dependencies)
}

func (c FeedConverter) toHcl(resource octopus2.Feed, _ bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllFeeds, c.ExcludeFeeds, c.ExcludeFeedsRegex, c.ExcludeFeedsExcept) {
		return nil
	}

	forceLookup := lookup ||
		strutil.EmptyIfNil(resource.FeedType) == "BuiltIn" ||
		strutil.EmptyIfNil(resource.FeedType) == "OctopusProject"

	resourceName := "feed_" + sanitizer.SanitizeName(resource.Name)

	thisResource := data.ResourceDetails{}

	thisResource.Name = resource.Name
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()

	if forceLookup {
		c.toHclLookup(resource, &thisResource, resourceName)
	} else {
		c.toHclResource(stateless, dependencies, resource, &thisResource, resourceName)
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c FeedConverter) toHclResource(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) {
	if !(c.exportProjectFeed(resource) ||
		c.exportDocker(stateless, dependencies, resource, thisResource, resourceName) ||
		c.exportAws(stateless, dependencies, resource, thisResource, resourceName) ||
		c.exportMaven(stateless, dependencies, resource, thisResource, resourceName) ||
		c.exportGithub(stateless, dependencies, resource, thisResource, resourceName) ||
		c.exportHelm(stateless, dependencies, resource, thisResource, resourceName) ||
		c.exportNuget(stateless, dependencies, resource, thisResource, resourceName)) {
		zap.L().Error("Found unexpected feed type \"" + strutil.EmptyIfNil(resource.FeedType) + "\" with name \"" + resource.Name + "\".")
	}
}

func (c FeedConverter) exportProjectFeed(resource octopus2.Feed) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "OctopusProject" {
		return true
	}

	return false
}

func (c FeedConverter) exportDocker(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "Docker" {

		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
				"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
				": " + octopusdeployDockerContainerRegistryResourceType + "." + resourceName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployDockerContainerRegistryResourceType + "." + resourceName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeployDockerContainerRegistryResourceType + "." + resourceName + ".id}"
		}

		passwordName := resourceName + "_password"

		parameters := []data.ResourceParameter{}
		if resource.Password != nil && resource.Password.HasValue {

			parameters = append(parameters, data.ResourceParameter{
				Label:         "Docker Feed " + resource.Name + " password",
				Description:   "The password associated with the feed \"" + resource.Name + "\"",
				ResourceName:  sanitizer.SanitizeParameterName(dependencies, resource.Name, "Password"),
				ParameterType: "Password",
				Sensitive:     true,
				VariableName:  passwordName,
			})
		}

		thisResource.Parameters = parameters
		thisResource.ToHcl = func() (string, error) {

			password := "${var." + passwordName + "}"
			terraformResource := terraform2.TerraformDockerFeed{
				Type:                              octopusdeployDockerContainerRegistryResourceType,
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				RegistryPath:                      resource.RegistryPath,
				Username:                          strutil.NilIfEmptyPointer(resource.Username),
				ApiVersion:                        resource.ApiVersion,
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
				FeedUri:                           resource.FeedUri,
			}

			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, resource.Name, "Docker", resourceName)
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 ? 0 : 1}")
			}

			if resource.Password != nil && resource.Password.HasValue {
				secretVariableResource := terraform2.TerraformVariable{
					Name:        passwordName,
					Type:        "string",
					Nullable:    false,
					Sensitive:   true,
					Description: "The password used by the feed " + resource.Name,
				}

				terraformResource.Password = &password

				if c.DummySecretVariableValues {
					secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				}

				block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
				hcl.WriteUnquotedAttribute(block, "type", "string")
				file.Body().AppendBlock(block)
			}

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues || stateless {

				ignoreAll := terraform2.EmptyBlock{}
				lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
				targetBlock.Body().AppendBlock(lifecycleBlock)

				if c.DummySecretVariableValues {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "[password]")
				}

				if stateless {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
				}
			}

			file.Body().AppendBlock(targetBlock)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) exportAws(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "AwsElasticContainerRegistry" {
		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
				"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
				": " + octopusdeployAwsElasticContainerRegistryResourceType + "." + resourceName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployAwsElasticContainerRegistryResourceType + "." + resourceName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeployAwsElasticContainerRegistryResourceType + "." + resourceName + ".id}"
		}

		passwordName := resourceName + "_password"

		thisResource.Parameters = []data.ResourceParameter{
			{
				Label:         "ECR Feed " + resource.Name + " password",
				Description:   "The password associated with the feed \"" + resource.Name + "\"",
				ResourceName:  sanitizer.SanitizeParameterName(dependencies, resource.Name, "Password"),
				ParameterType: "Password",
				Sensitive:     true,
				VariableName:  passwordName,
			},
		}
		thisResource.ToHcl = func() (string, error) {

			password := "${var." + passwordName + "}"
			terraformResource := terraform2.TerraformEcrFeed{
				Type:                              octopusdeployAwsElasticContainerRegistryResourceType,
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				AccessKey:                         resource.AccessKey,
				SecretKey:                         &password,
				Region:                            resource.Region,
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
			}

			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, resource.Name, "AwsElasticContainerRegistry", resourceName)
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 ? 0 : 1}")
			}

			if resource.SecretKey != nil && resource.SecretKey.HasValue {
				secretVariableResource := terraform2.TerraformVariable{
					Name:        passwordName,
					Type:        "string",
					Nullable:    false,
					Sensitive:   true,
					Description: "The secret key used by the feed " + resource.Name,
				}

				terraformResource.SecretKey = &password

				if c.DummySecretVariableValues {
					secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				}

				block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
				hcl.WriteUnquotedAttribute(block, "type", "string")
				file.Body().AppendBlock(block)
			}

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues || stateless {

				ignoreAll := terraform2.EmptyBlock{}
				lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
				targetBlock.Body().AppendBlock(lifecycleBlock)

				if c.DummySecretVariableValues {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "[secret_key]")
				}

				if stateless {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
				}
			}

			file.Body().AppendBlock(targetBlock)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) exportMaven(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "Maven" {
		thisResource.Lookup = "${" + octopusdeployMavenFeedResourceType + "." + resourceName + ".id}"

		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
				"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
				": " + octopusdeployMavenFeedResourceType + "." + resourceName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployMavenFeedResourceType + "." + resourceName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeployMavenFeedResourceType + "." + resourceName + ".id}"
		}

		passwordName := resourceName + "_password"

		parameters := []data.ResourceParameter{}

		if resource.Password != nil && resource.Password.HasValue {

			parameters = append(parameters, data.ResourceParameter{
				Label:         "Maven Feed " + resource.Name + " password",
				Description:   "The password associated with the feed \"" + resource.Name + "\"",
				ResourceName:  sanitizer.SanitizeParameterName(dependencies, resource.Name, "Password"),
				ParameterType: "Password",
				Sensitive:     true,
				VariableName:  passwordName,
			})
		}

		thisResource.Parameters = parameters
		thisResource.ToHcl = func() (string, error) {
			password := "${var." + passwordName + "}"
			terraformResource := terraform2.TerraformMavenFeed{
				Type:                              octopusdeployMavenFeedResourceType,
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				FeedUri:                           resource.FeedUri,
				Username:                          strutil.NilIfEmptyPointer(resource.Username),
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
				DownloadAttempts:                  resource.DownloadAttempts,
				DownloadRetryBackoffSeconds:       resource.DownloadRetryBackoffSeconds,
			}

			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, resource.Name, "Maven", resourceName)
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 ? 0 : 1}")
			}

			if resource.Password != nil && resource.Password.HasValue {
				secretVariableResource := terraform2.TerraformVariable{
					Name:        passwordName,
					Type:        "string",
					Nullable:    false,
					Sensitive:   true,
					Description: "The password used by the feed " + resource.Name,
				}

				terraformResource.Password = &password

				if c.DummySecretVariableValues {
					secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				}

				block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
				hcl.WriteUnquotedAttribute(block, "type", "string")
				file.Body().AppendBlock(block)
			}

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues || stateless {

				ignoreAll := terraform2.EmptyBlock{}
				lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
				targetBlock.Body().AppendBlock(lifecycleBlock)

				if c.DummySecretVariableValues {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "[password]")
				}

				if stateless {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
				}
			}

			file.Body().AppendBlock(targetBlock)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) exportGithub(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "GitHub" {
		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
				"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
				": " + octopusdeployGithubRepositoryFeedResourceType + "." + resourceName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployGithubRepositoryFeedResourceType + "." + resourceName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeployGithubRepositoryFeedResourceType + "." + resourceName + ".id}"
		}

		passwordName := resourceName + "_password"

		parameters := []data.ResourceParameter{}
		if resource.Password != nil && resource.Password.HasValue {
			parameters = append(parameters, data.ResourceParameter{
				Label:         "GitHub Feed " + resource.Name + " password",
				Description:   "The password associated with the feed \"" + resource.Name + "\"",
				ResourceName:  sanitizer.SanitizeParameterName(dependencies, resource.Name, "Password"),
				ParameterType: "Password",
				Sensitive:     true,
				VariableName:  passwordName,
			})
		}

		thisResource.Parameters = parameters
		thisResource.ToHcl = func() (string, error) {

			password := "${var." + passwordName + "}"
			terraformResource := terraform2.TerraformGitHubRepoFeed{
				Type:                              octopusdeployGithubRepositoryFeedResourceType,
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				FeedUri:                           resource.FeedUri,
				Username:                          strutil.NilIfEmptyPointer(resource.Username),
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
				DownloadAttempts:                  resource.DownloadAttempts,
				DownloadRetryBackoffSeconds:       resource.DownloadRetryBackoffSeconds,
			}

			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, resource.Name, "GitHub", resourceName)
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 ? 0 : 1}")
			}

			if resource.Password != nil && resource.Password.HasValue {
				secretVariableResource := terraform2.TerraformVariable{
					Name:        passwordName,
					Type:        "string",
					Nullable:    false,
					Sensitive:   true,
					Description: "The password used by the feed " + resource.Name,
				}

				terraformResource.Password = &password

				if c.DummySecretVariableValues {
					secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				}

				block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
				hcl.WriteUnquotedAttribute(block, "type", "string")
				file.Body().AppendBlock(block)
			}

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues || stateless {

				ignoreAll := terraform2.EmptyBlock{}
				lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
				targetBlock.Body().AppendBlock(lifecycleBlock)

				if c.DummySecretVariableValues {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "[password]")
				}

				if stateless {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
				}
			}

			file.Body().AppendBlock(targetBlock)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) exportHelm(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "Helm" {
		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
				"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
				": " + octopusdeployHelmFeedResourceType + "." + resourceName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployHelmFeedResourceType + "." + resourceName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeployHelmFeedResourceType + "." + resourceName + ".id}"
		}

		passwordName := resourceName + "_password"

		parameters := []data.ResourceParameter{}
		if resource.Password != nil && resource.Password.HasValue {
			parameters = append(parameters, data.ResourceParameter{
				Label:         "Helm Feed " + resource.Name + " password",
				Description:   "The password associated with the feed \"" + resource.Name + "\"",
				ResourceName:  sanitizer.SanitizeParameterName(dependencies, resource.Name, "Password"),
				ParameterType: "Password",
				Sensitive:     true,
				VariableName:  passwordName,
			})
		}

		thisResource.Parameters = parameters
		thisResource.ToHcl = func() (string, error) {

			password := "${var." + passwordName + "}"

			terraformResource := terraform2.TerraformHelmFeed{
				Type:                              octopusdeployHelmFeedResourceType,
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				FeedUri:                           resource.FeedUri,
				Username:                          strutil.NilIfEmptyPointer(resource.Username),
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
			}

			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, resource.Name, "Helm", resourceName)
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 ? 0 : 1}")
			}

			if resource.Password != nil && resource.Password.HasValue {
				secretVariableResource := terraform2.TerraformVariable{
					Name:        passwordName,
					Type:        "string",
					Nullable:    false,
					Sensitive:   true,
					Description: "The password used by the feed " + resource.Name,
				}

				terraformResource.Password = &password

				if c.DummySecretVariableValues {
					secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				}

				block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
				hcl.WriteUnquotedAttribute(block, "type", "string")
				file.Body().AppendBlock(block)
			}

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues || stateless {

				ignoreAll := terraform2.EmptyBlock{}
				lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
				targetBlock.Body().AppendBlock(lifecycleBlock)

				if c.DummySecretVariableValues {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "[password]")
				}

				if stateless {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
				}
			}

			file.Body().AppendBlock(targetBlock)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) exportNuget(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "NuGet" {
		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
				"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
				": " + octopusdeploy_nuget_feed_resource_type + "." + resourceName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeploy_nuget_feed_resource_type + "." + resourceName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeploy_nuget_feed_resource_type + "." + resourceName + ".id}"
		}

		passwordName := resourceName + "_password"

		thisResource.Parameters = []data.ResourceParameter{
			{
				Label:         "Nuget Feed " + resource.Name + " password",
				Description:   "The password associated with the feed \"" + resource.Name + "\"",
				ResourceName:  sanitizer.SanitizeParameterName(dependencies, resource.Name, "Password"),
				ParameterType: "Password",
				Sensitive:     true,
				VariableName:  passwordName,
			},
		}
		thisResource.ToHcl = func() (string, error) {
			password := "${var." + passwordName + "}"

			terraformResource := terraform2.TerraformNuGetFeed{
				Type:                              octopusdeploy_nuget_feed_resource_type,
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				FeedUri:                           resource.FeedUri,
				Username:                          strutil.NilIfEmptyPointer(resource.Username),
				IsEnhancedMode:                    resource.EnhancedMode,
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
				DownloadAttempts:                  resource.DownloadAttempts,
				DownloadRetryBackoffSeconds:       resource.DownloadRetryBackoffSeconds,
			}

			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, resource.Name, "NuGet", resourceName)
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 ? 0 : 1}")
			}

			if resource.Password != nil && resource.Password.HasValue {
				secretVariableResource := terraform2.TerraformVariable{
					Name:        passwordName,
					Type:        "string",
					Nullable:    false,
					Sensitive:   true,
					Description: "The password used by the feed " + resource.Name,
				}

				terraformResource.Password = &password

				if c.DummySecretVariableValues {
					secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				}

				block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
				hcl.WriteUnquotedAttribute(block, "type", "string")
				file.Body().AppendBlock(block)
			}

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues || stateless {

				ignoreAll := terraform2.EmptyBlock{}
				lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
				targetBlock.Body().AppendBlock(lifecycleBlock)

				if c.DummySecretVariableValues {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "[password]")
				}

				if stateless {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
				}
			}

			file.Body().AppendBlock(targetBlock)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) toHclLookup(resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) {
	thisResource.Lookup = "${data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id}"

	if !(c.lookupBuiltIn(resource, thisResource, resourceName) ||
		c.lookupDocker(resource, thisResource, resourceName) ||
		c.lookupAws(resource, thisResource, resourceName) ||
		c.lookupMaven(resource, thisResource, resourceName) ||
		c.lookupGithub(resource, thisResource, resourceName) ||
		c.lookupHelm(resource, thisResource, resourceName) ||
		c.lookupNuget(resource, thisResource, resourceName) ||
		c.lookupOctopusProject(resource, thisResource, resourceName)) {
		zap.L().Error("Found unexpected feed type \"" + strutil.EmptyIfNil(resource.FeedType) + "\" with name \"" + resource.Name + "\".")
	}
}

func (c FeedConverter) lookupOctopusProject(resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "OctopusProject" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData(resourceName, "", "OctopusProject")
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) lookupNuget(resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "NuGet" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData(resourceName, resource.Name, "NuGet")
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) lookupHelm(resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "Helm" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData(resourceName, resource.Name, "Helm")
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) lookupGithub(resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "GitHub" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData(resourceName, resource.Name, "GitHub")
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) lookupMaven(resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "Maven" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData(resourceName, resource.Name, "Maven")
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) lookupAws(resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "AwsElasticContainerRegistry" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData(resourceName, resource.Name, "AwsElasticContainerRegistry")
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) lookupDocker(resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "Docker" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData(resourceName, resource.Name, "Docker")
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) lookupBuiltIn(resource octopus2.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "BuiltIn" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData(resourceName, "", "BuiltIn")
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \"BuiltIn\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) buildData(resourceName string, partialName string, feedType string) terraform2.TerraformFeedData {
	return terraform2.TerraformFeedData{
		Type:        octopusdeployFeedsDataType,
		Name:        resourceName,
		PartialName: partialName,
		FeedType:    feedType,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c FeedConverter) writeData(file *hclwrite.File, partialName string, feedType string, resourceName string) {
	terraformResource := c.buildData(resourceName, partialName, feedType)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}
