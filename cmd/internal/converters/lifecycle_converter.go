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

const octopusdeployLifecyclesDataType = "octopusdeploy_lifecycles"
const octopusdeployLifecycleResourceType = "octopusdeploy_lifecycle"

type LifecycleConverter struct {
	Client                   client.OctopusClient
	EnvironmentConverter     ConverterAndLookupWithStatelessById
	ErrGroup                 *errgroup.Group
	ExcludeLifecycles        args.StringSliceArgs
	ExcludeLifecyclesRegex   args.StringSliceArgs
	ExcludeLifecyclesExcept  args.StringSliceArgs
	ExcludeAllLifecycles     bool
	Excluder                 ExcludeByName
	LimitResourceCount       int
	IncludeSpaceInPopulation bool
	IncludeIds               bool
}

func (c LifecycleConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c LifecycleConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c LifecycleConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllLifecycles {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.Lifecycle]{
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
		if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllLifecycles, c.ExcludeLifecycles, c.ExcludeLifecyclesRegex, c.ExcludeLifecyclesExcept) {
			continue
		}

		zap.L().Info("Lifecycle: " + resource.Id)
		err := c.toHcl(resource, false, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c LifecycleConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c LifecycleConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c LifecycleConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Channels can have empty strings for the lifecycle ID
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Lifecycle{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllLifecycles, c.ExcludeLifecycles, c.ExcludeLifecyclesRegex, c.ExcludeLifecyclesExcept) {
		return nil
	}

	zap.L().Info("Lifecycle: " + resource.Id)
	return c.toHcl(resource, true, false, stateless, dependencies)

}

func (c LifecycleConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	// Channels can have empty strings for the lifecycle ID
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	lifecycle := octopus.Lifecycle{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &lifecycle)

	if err != nil {
		return err
	}

	if c.Excluder.IsResourceExcludedWithRegex(lifecycle.Name, c.ExcludeAllLifecycles, c.ExcludeLifecycles, c.ExcludeLifecyclesRegex, c.ExcludeLifecyclesExcept) {
		return nil
	}

	return c.toHcl(lifecycle, false, true, false, dependencies)

}

func (c LifecycleConverter) buildData(resourceName string, resource octopus.Lifecycle) terraform.TerraformLifecycleData {
	return terraform.TerraformLifecycleData{
		Type:        octopusdeployLifecyclesDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c LifecycleConverter) writeData(file *hclwrite.File, resource octopus.Lifecycle, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

// toBashImport creates a bash script to import the resource
func (c LifecycleConverter) toBashImport(resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Lifecycles" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No lifecycle found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing lifecycle ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, octopusdeployLifecycleResourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c LifecycleConverter) toPowershellImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Lifecycles?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No lifecycle found with the name $ResourceName"
	exit 1
}

echo "Importing lifecycle $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, octopusdeployLifecycleResourceType, resourceName), nil
		},
	})
}

func (c LifecycleConverter) toHcl(lifecycle octopus.Lifecycle, recursive bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {

	if c.Excluder.IsResourceExcludedWithRegex(lifecycle.Name, c.ExcludeAllLifecycles, c.ExcludeLifecycles, c.ExcludeLifecyclesRegex, c.ExcludeLifecyclesExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + lifecycle.Id)
		return nil
	}

	if recursive {
		// The environments are a dependency that we need to lookup
		for _, phase := range lifecycle.Phases {
			for _, auto := range phase.AutomaticDeploymentTargets {
				if stateless {
					err := c.EnvironmentConverter.ToHclStatelessById(auto, dependencies)
					if err != nil {
						return err
					}
				} else {
					err := c.EnvironmentConverter.ToHclById(auto, dependencies)
					if err != nil {
						return err
					}
				}

			}
			for _, optional := range phase.OptionalDeploymentTargets {
				if stateless {
					err := c.EnvironmentConverter.ToHclStatelessById(optional, dependencies)

					if err != nil {
						return err
					}
				} else {
					err := c.EnvironmentConverter.ToHclById(optional, dependencies)

					if err != nil {
						return err
					}
				}

			}
		}
	}

	forceLookup := lookup || lifecycle.Name == "Default Lifecycle"

	resourceName := "lifecycle_" + sanitizer.SanitizeName(lifecycle.Name)

	c.toBashImport(resourceName, lifecycle.Name, dependencies)
	c.toPowershellImport(resourceName, lifecycle.Name, dependencies)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = lifecycle.Id
	thisResource.Name = lifecycle.Name
	thisResource.ResourceType = c.GetResourceType()
	if forceLookup {
		thisResource.Lookup = "${data." + octopusdeployLifecyclesDataType + "." + resourceName + ".lifecycles[0].id}"

		thisResource.ToHcl = func() (string, error) {
			data := c.buildData(resourceName, lifecycle)
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(data, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a lifecycle called \""+lifecycle.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.lifecycles) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	} else {
		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployLifecyclesDataType + "." + resourceName + ".lifecycles) != 0 " +
				"? data." + octopusdeployLifecyclesDataType + "." + resourceName + ".lifecycles[0].id " +
				": " + octopusdeployLifecycleResourceType + "." + resourceName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployLifecycleResourceType + "." + resourceName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeployLifecycleResourceType + "." + resourceName + ".id}"
		}

		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform.TerraformLifecycle{
				Type:                    octopusdeployLifecycleResourceType,
				Name:                    resourceName,
				Id:                      strutil.InputPointerIfEnabled(c.IncludeIds, &lifecycle.Id),
				SpaceId:                 strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", lifecycle.SpaceId)),
				ResourceName:            lifecycle.Name,
				Description:             lifecycle.Description,
				Phase:                   c.convertPhases(lifecycle.Phases, dependencies),
				ReleaseRetentionPolicy:  c.convertPolicy(lifecycle.ReleaseRetentionPolicy),
				TentacleRetentionPolicy: c.convertPolicy(lifecycle.TentacleRetentionPolicy),
			}
			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, lifecycle, resourceName)
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployLifecyclesDataType + "." + resourceName + ".lifecycles) != 0 ? 0 : 1}")
			}

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			if stateless {
				hcl.WriteLifecyclePreventDestroyAttribute(block)
			}

			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c LifecycleConverter) GetResourceType() string {
	return "Lifecycles"
}

func (c LifecycleConverter) convertPolicy(policy *octopus.Policy) *terraform.TerraformPolicy {
	if policy == nil {
		return nil
	}

	return &terraform.TerraformPolicy{
		QuantityToKeep:    policy.QuantityToKeep,
		ShouldKeepForever: policy.ShouldKeepForever,
		Unit:              policy.Unit,
	}
}

func (c LifecycleConverter) convertPhases(phases []octopus.Phase, dependencies *data.ResourceDetailsCollection) []terraform.TerraformPhase {
	terraformPhases := make([]terraform.TerraformPhase, 0)
	for _, v := range phases {
		terraformPhases = append(terraformPhases, terraform.TerraformPhase{
			AutomaticDeploymentTargets:         c.convertTargets(v.AutomaticDeploymentTargets, dependencies),
			OptionalDeploymentTargets:          c.convertTargets(v.OptionalDeploymentTargets, dependencies),
			Name:                               v.Name,
			IsOptionalPhase:                    v.IsOptionalPhase,
			MinimumEnvironmentsBeforePromotion: v.MinimumEnvironmentsBeforePromotion,
			ReleaseRetentionPolicy:             c.convertPolicy(v.ReleaseRetentionPolicy),
			TentacleRetentionPolicy:            c.convertPolicy(v.TentacleRetentionPolicy),
		})
	}
	return terraformPhases
}

func (c LifecycleConverter) convertTargets(environments []string, dependencies *data.ResourceDetailsCollection) []string {
	converted := make([]string, len(environments))

	for i, v := range environments {
		converted[i] = dependencies.GetResource("Environments", v)
	}

	return converted
}
