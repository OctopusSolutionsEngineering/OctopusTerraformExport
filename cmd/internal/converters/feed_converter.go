package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/boolutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/naming"
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
const octopusdeployNugetFeedResourceType = "octopusdeploy_nuget_feed"
const octopusdeployArtifactoryFeedResourceType = "octopusdeploy_artifactory_generic_feed"
const octopusdeployS3FeedResourceType = "octopusdeploy_s3_feed"

const artifactory_feed_type = "ArtifactoryGeneric"
const s3_feed_type = "S3"

type FeedConverter struct {
	Client                    client.OctopusClient
	DummySecretVariableValues bool
	DummySecretGenerator      dummy.DummySecretGenerator
	ErrGroup                  *errgroup.Group
	ExcludeFeeds              args.StringSliceArgs
	ExcludeFeedsRegex         args.StringSliceArgs
	ExcludeFeedsExcept        args.StringSliceArgs
	ExcludeAllFeeds           bool
	Excluder                  ExcludeByName
	IncludeIds                bool
	LimitResourceCount        int
	IncludeSpaceInPopulation  bool
	GenerateImportScripts     bool
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
	if c.ExcludeAllFeeds {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.Feed]{
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
		if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllFeeds, c.ExcludeFeeds, c.ExcludeFeedsRegex, c.ExcludeFeedsExcept) {
			continue
		}

		zap.L().Info("Feed: " + resource.Id + " " + resource.Name)
		err := c.toHcl(resource, false, false, stateless, dependencies)

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

	resource := octopus.Feed{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Feed: %w", err)
	}

	zap.L().Info("Feed: " + resource.Id + " " + resource.Name)
	return c.toHcl(resource, true, false, stateless, dependencies)
}

func (c FeedConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Feed{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Feed: %w", err)
	}

	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllFeeds, c.ExcludeFeeds, c.ExcludeFeedsRegex, c.ExcludeFeedsExcept) {
		return nil
	}

	return c.toHcl(resource, false, true, false, dependencies)
}

// toBashImport creates a bash script to import the resource
func (c FeedConverter) toBashImport(resourceType string, resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Feeds" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No feed found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing feed ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, resourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c FeedConverter) toPowershellImport(resourceType string, resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Feeds?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No feed found with the name $ResourceName"
	exit 1
}

echo "Importing feed $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, octopusResourceName, resourceType, resourceName), nil
		},
	})
}

func (c FeedConverter) toHcl(resource octopus.Feed, _ bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllFeeds, c.ExcludeFeeds, c.ExcludeFeedsRegex, c.ExcludeFeedsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + resource.Id)
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

func (c FeedConverter) toHclResource(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) {
	if !(c.exportProjectFeed(resource) ||
		c.exportDocker(stateless, dependencies, resource, thisResource, resourceName) ||
		c.exportAws(stateless, dependencies, resource, thisResource, resourceName) ||
		c.exportMaven(stateless, dependencies, resource, thisResource, resourceName) ||
		c.exportGithub(stateless, dependencies, resource, thisResource, resourceName) ||
		c.exportHelm(stateless, dependencies, resource, thisResource, resourceName) ||
		c.exportArtifactory(stateless, dependencies, resource, thisResource, resourceName) ||
		c.exportS3(stateless, dependencies, resource, thisResource, resourceName) ||
		c.exportNuget(stateless, dependencies, resource, thisResource, resourceName)) {
		zap.L().Error("Found unexpected feed type \"" + strutil.EmptyIfNil(resource.FeedType) + "\" with name \"" + resource.Name + "\".")
	}
}

func (c FeedConverter) exportProjectFeed(resource octopus.Feed) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "OctopusProject" {
		return true
	}

	return false
}

func (c FeedConverter) exportDocker(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) != "Docker" {
		return false
	}

	if c.GenerateImportScripts {
		c.toBashImport(octopusdeployDockerContainerRegistryResourceType, resourceName, resource.Name, dependencies)
		c.toPowershellImport(octopusdeployDockerContainerRegistryResourceType, resourceName, resource.Name, dependencies)
	}

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
			"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
			": " + octopusdeployDockerContainerRegistryResourceType + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployDockerContainerRegistryResourceType + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployDockerContainerRegistryResourceType + "." + resourceName + ".id}"
	}

	passwordName := naming.FeedSecretName(resource)

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
		terraformResource := terraform.TerraformDockerFeed{
			Type:                              octopusdeployDockerContainerRegistryResourceType,
			Name:                              resourceName,
			Id:                                strutil.InputPointerIfEnabled(c.IncludeIds, &resource.Id),
			SpaceId:                           strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", resource.SpaceId)),
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
			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			terraformResource.Password = &password

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				dependencies.AddDummy(data.DummyVariableReference{
					VariableName: passwordName,
					ResourceName: resource.Name,
					ResourceType: c.GetResourceType(),
				})
			}

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)
		}

		targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		c.writeLifecycleAttributes(targetBlock, "password", stateless)

		file.Body().AppendBlock(targetBlock)

		return string(file.Bytes()), nil
	}

	return true
}

func (c FeedConverter) exportAws(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) != "AwsElasticContainerRegistry" {
		return false
	}

	if c.GenerateImportScripts {
		c.toBashImport(octopusdeployAwsElasticContainerRegistryResourceType, resourceName, resource.Name, dependencies)
		c.toPowershellImport(octopusdeployAwsElasticContainerRegistryResourceType, resourceName, resource.Name, dependencies)
	}

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
			"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
			": " + octopusdeployAwsElasticContainerRegistryResourceType + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployAwsElasticContainerRegistryResourceType + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployAwsElasticContainerRegistryResourceType + "." + resourceName + ".id}"
	}

	passwordName := naming.FeedSecretKeyName(resource)

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
		terraformResource := terraform.TerraformEcrFeed{
			Type:                              octopusdeployAwsElasticContainerRegistryResourceType,
			Name:                              resourceName,
			Id:                                strutil.InputPointerIfEnabled(c.IncludeIds, &resource.Id),
			SpaceId:                           strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", resource.SpaceId)),
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
			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The secret key used by the feed " + resource.Name,
			}

			terraformResource.SecretKey = &password

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				dependencies.AddDummy(data.DummyVariableReference{
					VariableName: passwordName,
					ResourceName: resource.Name,
					ResourceType: c.GetResourceType(),
				})
			}

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)
		}

		targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		c.writeLifecycleAttributes(targetBlock, "secret_key", stateless)

		file.Body().AppendBlock(targetBlock)

		return string(file.Bytes()), nil
	}

	return true
}

func (c FeedConverter) exportMaven(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) != "Maven" {
		return false
	}

	if c.GenerateImportScripts {
		c.toBashImport(octopusdeployMavenFeedResourceType, resourceName, resource.Name, dependencies)
		c.toPowershellImport(octopusdeployMavenFeedResourceType, resourceName, resource.Name, dependencies)
	}

	thisResource.Lookup = "${" + octopusdeployMavenFeedResourceType + "." + resourceName + ".id}"

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
			"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
			": " + octopusdeployMavenFeedResourceType + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployMavenFeedResourceType + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployMavenFeedResourceType + "." + resourceName + ".id}"
	}

	passwordName := naming.FeedSecretName(resource)

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
		terraformResource := terraform.TerraformMavenFeed{
			Type:                              octopusdeployMavenFeedResourceType,
			Id:                                strutil.InputPointerIfEnabled(c.IncludeIds, &resource.Id),
			SpaceId:                           strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", resource.SpaceId)),
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
			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			terraformResource.Password = &password

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				dependencies.AddDummy(data.DummyVariableReference{
					VariableName: passwordName,
					ResourceName: resource.Name,
					ResourceType: c.GetResourceType(),
				})
			}

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)
		}

		targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		c.writeLifecycleAttributes(targetBlock, "password", stateless)

		file.Body().AppendBlock(targetBlock)

		return string(file.Bytes()), nil
	}

	return true
}

func (c FeedConverter) exportGithub(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) != "GitHub" {
		return false
	}

	if c.GenerateImportScripts {
		c.toBashImport(octopusdeployGithubRepositoryFeedResourceType, resourceName, resource.Name, dependencies)
		c.toPowershellImport(octopusdeployGithubRepositoryFeedResourceType, resourceName, resource.Name, dependencies)
	}

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
			"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
			": " + octopusdeployGithubRepositoryFeedResourceType + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployGithubRepositoryFeedResourceType + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployGithubRepositoryFeedResourceType + "." + resourceName + ".id}"
	}

	passwordName := naming.FeedSecretName(resource)

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
		terraformResource := terraform.TerraformGitHubRepoFeed{
			Type:                              octopusdeployGithubRepositoryFeedResourceType,
			Id:                                strutil.InputPointerIfEnabled(c.IncludeIds, &resource.Id),
			SpaceId:                           strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", resource.SpaceId)),
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
			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			terraformResource.Password = &password

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				dependencies.AddDummy(data.DummyVariableReference{
					VariableName: passwordName,
					ResourceName: resource.Name,
					ResourceType: c.GetResourceType(),
				})
			}

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)
		}

		targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		c.writeLifecycleAttributes(targetBlock, "password", stateless)

		file.Body().AppendBlock(targetBlock)

		return string(file.Bytes()), nil
	}

	return true

}

func (c FeedConverter) exportHelm(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) != "Helm" {
		return false
	}

	if c.GenerateImportScripts {
		c.toBashImport(octopusdeployHelmFeedResourceType, resourceName, resource.Name, dependencies)
		c.toPowershellImport(octopusdeployHelmFeedResourceType, resourceName, resource.Name, dependencies)
	}

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
			"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
			": " + octopusdeployHelmFeedResourceType + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployHelmFeedResourceType + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployHelmFeedResourceType + "." + resourceName + ".id}"
	}

	passwordName := naming.FeedSecretName(resource)

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

		terraformResource := terraform.TerraformHelmFeed{
			Type:                              octopusdeployHelmFeedResourceType,
			Id:                                strutil.InputPointerIfEnabled(c.IncludeIds, &resource.Id),
			SpaceId:                           strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", resource.SpaceId)),
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
			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			terraformResource.Password = &password

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				dependencies.AddDummy(data.DummyVariableReference{
					VariableName: passwordName,
					ResourceName: resource.Name,
					ResourceType: c.GetResourceType(),
				})
			}

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)
		}

		targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		c.writeLifecycleAttributes(targetBlock, "password", stateless)

		file.Body().AppendBlock(targetBlock)

		return string(file.Bytes()), nil
	}

	return true

}

func (c FeedConverter) exportNuget(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) != "NuGet" {
		return false
	}

	if c.GenerateImportScripts {
		c.toBashImport(octopusdeployNugetFeedResourceType, resourceName, resource.Name, dependencies)
		c.toPowershellImport(octopusdeployNugetFeedResourceType, resourceName, resource.Name, dependencies)
	}

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
			"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
			": " + octopusdeployNugetFeedResourceType + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployNugetFeedResourceType + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployNugetFeedResourceType + "." + resourceName + ".id}"
	}

	passwordName := naming.FeedSecretName(resource)

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

		terraformResource := terraform.TerraformNuGetFeed{
			Type:                              octopusdeployNugetFeedResourceType,
			Id:                                strutil.InputPointerIfEnabled(c.IncludeIds, &resource.Id),
			SpaceId:                           strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", resource.SpaceId)),
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
			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			terraformResource.Password = &password

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				dependencies.AddDummy(data.DummyVariableReference{
					VariableName: passwordName,
					ResourceName: resource.Name,
					ResourceType: c.GetResourceType(),
				})
			}

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)
		}

		targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		c.writeLifecycleAttributes(targetBlock, "password", stateless)

		file.Body().AppendBlock(targetBlock)

		return string(file.Bytes()), nil
	}

	return true
}

func (c FeedConverter) exportArtifactory(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) != artifactory_feed_type {
		return false
	}

	if c.GenerateImportScripts {
		c.toBashImport(octopusdeployArtifactoryFeedResourceType, resourceName, resource.Name, dependencies)
		c.toPowershellImport(octopusdeployArtifactoryFeedResourceType, resourceName, resource.Name, dependencies)
	}

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
			"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
			": " + octopusdeployArtifactoryFeedResourceType + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployArtifactoryFeedResourceType + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployArtifactoryFeedResourceType + "." + resourceName + ".id}"
	}

	passwordName := naming.FeedSecretName(resource)

	thisResource.Parameters = []data.ResourceParameter{
		{
			Label:         "Artifactory Feed " + resource.Name + " password",
			Description:   "The password associated with the feed \"" + resource.Name + "\"",
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, resource.Name, "Password"),
			ParameterType: "Password",
			Sensitive:     true,
			VariableName:  passwordName,
		},
	}
	thisResource.ToHcl = func() (string, error) {
		password := "${var." + passwordName + "}"

		terraformResource := terraform.TerraformArtifactoryFeed{
			Type:         octopusdeployArtifactoryFeedResourceType,
			Name:         resourceName,
			Id:           strutil.InputPointerIfEnabled(c.IncludeIds, &resource.Id),
			FeedUri:      strutil.EmptyIfNil(resource.FeedUri),
			ResourceName: resource.Name,
			Password:     nil,
			Username:     strutil.NilIfEmptyPointer(resource.Username),
			Repository:   strutil.EmptyIfNil(resource.Repository),
			LayoutRegex:  resource.LayoutRegex,
		}

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, resource.Name, artifactory_feed_type, resourceName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 ? 0 : 1}")
		}

		if resource.Password != nil && resource.Password.HasValue {
			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			terraformResource.Password = &password

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				dependencies.AddDummy(data.DummyVariableReference{
					VariableName: passwordName,
					ResourceName: resource.Name,
					ResourceType: c.GetResourceType(),
				})
			}

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)
		}

		targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		c.writeLifecycleAttributes(targetBlock, "password", stateless)

		file.Body().AppendBlock(targetBlock)

		return string(file.Bytes()), nil
	}

	return true
}

func (c FeedConverter) exportS3(stateless bool, dependencies *data.ResourceDetailsCollection, resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) != s3_feed_type {
		return false
	}

	if c.GenerateImportScripts {
		c.toBashImport(octopusdeployS3FeedResourceType, resourceName, resource.Name, dependencies)
		c.toPowershellImport(octopusdeployS3FeedResourceType, resourceName, resource.Name, dependencies)
	}

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 " +
			"? data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id " +
			": " + octopusdeployS3FeedResourceType + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployS3FeedResourceType + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployS3FeedResourceType + "." + resourceName + ".id}"
	}

	passwordName := naming.FeedSecretName(resource)
	secretKeyName := naming.FeedSecretKeyName(resource)

	thisResource.Parameters = []data.ResourceParameter{
		{
			Label:         "S3 Feed " + resource.Name + " password",
			Description:   "The password associated with the feed \"" + resource.Name + "\"",
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, resource.Name, "Password"),
			ParameterType: "Password",
			Sensitive:     true,
			VariableName:  passwordName,
		},
	}
	thisResource.ToHcl = func() (string, error) {
		password := "${var." + passwordName + "}"
		secretKey := "${var." + secretKeyName + "}"

		terraformResource := terraform.TerraformS3Feed{
			Type:                  octopusdeployS3FeedResourceType,
			Name:                  resourceName,
			Id:                    strutil.InputPointerIfEnabled(c.IncludeIds, &resource.Id),
			Count:                 nil,
			ResourceName:          resource.Name,
			UseMachineCredentials: boolutil.FalseIfNil(resource.UseMachineCredentials),
			AccessKey:             resource.AccessKey,
			Password:              nil,
			SecretKey:             nil,
			Username:              strutil.NilIfEmptyPointer(resource.Username),
		}

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, resource.Name, s3_feed_type, resourceName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds) != 0 ? 0 : 1}")
		}

		if resource.Password != nil && resource.Password.HasValue {
			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password used by the feed " + resource.Name,
			}

			terraformResource.Password = &password

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				dependencies.AddDummy(data.DummyVariableReference{
					VariableName: passwordName,
					ResourceName: resource.Name,
					ResourceType: c.GetResourceType(),
				})
			}

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)
		}

		if resource.SecretKey != nil && resource.SecretKey.HasValue {
			secretVariableResource := terraform.TerraformVariable{
				Name:        secretKeyName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The secret key used by the feed " + resource.Name,
			}

			terraformResource.SecretKey = &secretKey

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				dependencies.AddDummy(data.DummyVariableReference{
					VariableName: passwordName,
					ResourceName: resource.Name,
					ResourceType: c.GetResourceType(),
				})
			}

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)
		}

		targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		c.writeLifecycleAttributes(targetBlock, "secret_key", stateless)

		file.Body().AppendBlock(targetBlock)

		return string(file.Bytes()), nil
	}

	return true
}

func (c FeedConverter) writeLifecycleAttributes(targetBlock *hclwrite.Block, secretProperty string, stateless bool) {
	// When using dummy values, we expect the secrets will be updated later
	if c.DummySecretVariableValues || stateless {

		ignoreAll := terraform.EmptyBlock{}
		lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
		targetBlock.Body().AppendBlock(lifecycleBlock)

		if c.DummySecretVariableValues {
			hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "["+secretProperty+"]")
		}

		if stateless {
			hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
		}
	}
}

func (c FeedConverter) toHclLookup(resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) {
	thisResource.Lookup = "${data." + octopusdeployFeedsDataType + "." + resourceName + ".feeds[0].id}"

	if !(c.lookupBuiltIn(resource, thisResource, resourceName) ||
		c.lookupDocker(resource, thisResource, resourceName) ||
		c.lookupAws(resource, thisResource, resourceName) ||
		c.lookupMaven(resource, thisResource, resourceName) ||
		c.lookupGithub(resource, thisResource, resourceName) ||
		c.lookupHelm(resource, thisResource, resourceName) ||
		c.lookupNuget(resource, thisResource, resourceName) ||
		c.lookupS3(resource, thisResource, resourceName) ||
		c.lookupArtifactory(resource, thisResource, resourceName) ||
		c.lookupOctopusProject(resource, thisResource, resourceName)) {
		zap.L().Error("Found unexpected feed type \"" + strutil.EmptyIfNil(resource.FeedType) + "\" with name \"" + resource.Name + "\".")
	}
}

func (c FeedConverter) lookupOctopusProject(resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == "OctopusProject" {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData(resourceName, "Octopus Server Releases", "OctopusProject")
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

func (c FeedConverter) lookupNuget(resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
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

func (c FeedConverter) lookupHelm(resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
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

func (c FeedConverter) lookupGithub(resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
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

func (c FeedConverter) lookupMaven(resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
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

func (c FeedConverter) lookupAws(resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
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

func (c FeedConverter) lookupDocker(resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
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

func (c FeedConverter) lookupS3(resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == artifactory_feed_type {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData(resourceName, resource.Name, artifactory_feed_type)
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

func (c FeedConverter) lookupArtifactory(resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
	if strutil.EmptyIfNil(resource.FeedType) == s3_feed_type {
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData(resourceName, resource.Name, s3_feed_type)
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

func (c FeedConverter) lookupBuiltIn(resource octopus.Feed, thisResource *data.ResourceDetails, resourceName string) bool {
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

func (c FeedConverter) buildData(resourceName string, partialName string, feedType string) terraform.TerraformFeedData {
	return terraform.TerraformFeedData{
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
