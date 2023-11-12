package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

type FeedConverter struct {
	Client                    client.OctopusClient
	DummySecretVariableValues bool
	DummySecretGenerator      DummySecretGenerator
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
		zap.L().Info("Feed: " + resource.Id)
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

	zap.L().Info("Feed: " + resource.Id)
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

func (c FeedConverter) toHcl(resource octopus2.Feed, _ bool, lookup bool, dependencies *ResourceDetailsCollection) error {
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
	if !(c.exportProjectFeed(resource) ||
		c.exportDocker(resource, thisResource, resourceName) ||
		c.exportAws(resource, thisResource, resourceName) ||
		c.exportMaven(resource, thisResource, resourceName) ||
		c.exportGithub(resource, thisResource, resourceName) ||
		c.exportHelm(resource, thisResource, resourceName) ||
		c.exportNuget(resource, thisResource, resourceName)) {
		zap.L().Error("Found unexpected feed type \"" + strutil.EmptyIfNil(resource.FeedType) + "\" with name \"" + resource.Name + "\".")
	}
}

func (c FeedConverter) exportProjectFeed(resource octopus2.Feed) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "OctopusProject" {
		return true
	}

	return false
}

func (c FeedConverter) exportDocker(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "Docker" {
		thisResource.Lookup = "${octopusdeploy_docker_container_registry." + resourceName + ".id}"
		thisResource.ToHcl = func() (string, error) {
			passwordName := resourceName + "_password"
			password := "${var." + passwordName + "}"
			terraformResource := terraform2.TerraformDockerFeed{
				Type:                              "octopusdeploy_docker_container_registry",
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				RegistryPath:                      resource.RegistryPath,
				Username:                          strutil.NilIfEmptyPointer(resource.Username),
				ApiVersion:                        resource.ApiVersion,
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
				FeedUri:                           resource.FeedUri,
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

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				hcl.WriteLifecycleAttribute(targetBlock, "[password]")
			}

			file.Body().AppendBlock(targetBlock)

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

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) exportAws(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "AwsElasticContainerRegistry" {
		thisResource.Lookup = "${octopusdeploy_aws_elastic_container_registry." + resourceName + ".id}"
		thisResource.ToHcl = func() (string, error) {
			passwordName := resourceName + "_password"
			password := "${var." + passwordName + "}"
			terraformResource := terraform2.TerraformEcrFeed{
				Type:                              "octopusdeploy_aws_elastic_container_registry",
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				AccessKey:                         resource.AccessKey,
				SecretKey:                         &password,
				Region:                            resource.Region,
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
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

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				hcl.WriteLifecycleAttribute(targetBlock, "[password]")
			}

			file.Body().AppendBlock(targetBlock)

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

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) exportMaven(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "Maven" {
		thisResource.Lookup = "${octopusdeploy_maven_feed." + resourceName + ".id}"
		thisResource.ToHcl = func() (string, error) {
			passwordName := resourceName + "_password"
			password := "${var." + passwordName + "}"
			terraformResource := terraform2.TerraformMavenFeed{
				Type:                              "octopusdeploy_maven_feed",
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				FeedUri:                           resource.FeedUri,
				Username:                          strutil.NilIfEmptyPointer(resource.Username),
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
				DownloadAttempts:                  resource.DownloadAttempts,
				DownloadRetryBackoffSeconds:       resource.DownloadRetryBackoffSeconds,
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

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				hcl.WriteLifecycleAttribute(targetBlock, "[password]")
			}

			file.Body().AppendBlock(targetBlock)

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

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) exportGithub(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "GitHub" {
		thisResource.Lookup = "${octopusdeploy_github_repository_feed." + resourceName + ".id}"
		thisResource.ToHcl = func() (string, error) {
			passwordName := resourceName + "_password"
			password := "${var." + passwordName + "}"

			terraformResource := terraform2.TerraformGitHubRepoFeed{
				Type:                              "octopusdeploy_github_repository_feed",
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				FeedUri:                           resource.FeedUri,
				Username:                          strutil.NilIfEmptyPointer(resource.Username),
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
				DownloadAttempts:                  resource.DownloadAttempts,
				DownloadRetryBackoffSeconds:       resource.DownloadRetryBackoffSeconds,
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

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				hcl.WriteLifecycleAttribute(targetBlock, "[password]")
			}

			file.Body().AppendBlock(targetBlock)

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

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) exportHelm(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "Helm" {
		thisResource.Lookup = "${octopusdeploy_helm_feed." + resourceName + ".id}"
		thisResource.ToHcl = func() (string, error) {
			passwordName := resourceName + "_password"
			password := "${var." + passwordName + "}"

			terraformResource := terraform2.TerraformHelmFeed{
				Type:                              "octopusdeploy_helm_feed",
				Name:                              resourceName,
				ResourceName:                      resource.Name,
				FeedUri:                           resource.FeedUri,
				Username:                          strutil.NilIfEmptyPointer(resource.Username),
				PackageAcquisitionLocationOptions: resource.PackageAcquisitionLocationOptions,
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

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				hcl.WriteLifecycleAttribute(targetBlock, "[password]")
			}

			file.Body().AppendBlock(targetBlock)

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

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) exportNuget(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "NuGet" {
		thisResource.Lookup = "${octopusdeploy_nuget_feed." + resourceName + ".id}"
		thisResource.ToHcl = func() (string, error) {

			passwordName := resourceName + "_password"
			password := "${var." + passwordName + "}"

			terraformResource := terraform2.TerraformNuGetFeed{
				Type:                              "octopusdeploy_nuget_feed",
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

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_nuget_feed." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				hcl.WriteLifecycleAttribute(targetBlock, "[password]")
			}

			file.Body().AppendBlock(targetBlock)

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

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) toHclLookup(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) {
	thisResource.Lookup = "${data.octopusdeploy_feeds." + resourceName + ".feeds[0].id}"

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

func (c FeedConverter) lookupOctopusProject(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "OctopusProject" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform2.TerraformFeedData{
				Type:     "octopusdeploy_feeds",
				Name:     resourceName,
				FeedType: "OctopusProject",
				Skip:     0,
				Take:     1,
			}
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

func (c FeedConverter) lookupNuget(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "NuGet" {
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
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) lookupHelm(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "Helm" {
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
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) lookupGithub(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "GitHub" {
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
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) lookupMaven(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "Maven" {
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
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) lookupAws(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "AwsElasticContainerRegistry" {
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
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) lookupDocker(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "Docker" {
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
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}

func (c FeedConverter) lookupBuiltIn(resource octopus2.Feed, thisResource *ResourceDetails, resourceName string) bool {
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
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a feed called \"BuiltIn\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.feeds) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		return true
	}

	return false
}
