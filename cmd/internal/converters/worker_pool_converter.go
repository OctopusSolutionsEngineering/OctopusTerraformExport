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

const octopusdeployWorkerPoolsDataType = "octopusdeploy_worker_pools"
const octopusdeployStaticWorkerPoolResourcePool = "octopusdeploy_static_worker_pool"
const octopusdeployDynamicWorkerPoolResourceType = "octopusdeploy_dynamic_worker_pool"

type WorkerPoolConverter struct {
	Client                   client.OctopusClient
	ErrGroup                 *errgroup.Group
	ExcludeWorkerpools       args.StringSliceArgs
	ExcludeWorkerpoolsRegex  args.StringSliceArgs
	ExcludeWorkerpoolsExcept args.StringSliceArgs
	ExcludeAllWorkerpools    bool
	Excluder                 ExcludeByName
	LimitResourceCount       int
	IncludeSpaceInPopulation bool
	IncludeIds               bool
	GenerateImportScripts    bool
}

func (c WorkerPoolConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c WorkerPoolConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c WorkerPoolConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllWorkerpools {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.WorkerPool]{
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
		if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllWorkerpools, c.ExcludeWorkerpools, c.ExcludeWorkerpoolsRegex, c.ExcludeWorkerpoolsExcept) {
			continue
		}

		zap.L().Info("Worker Pool: " + resource.Id + " " + resource.Name)
		err := c.toHcl(resource, false, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c WorkerPoolConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c WorkerPoolConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c WorkerPoolConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.WorkerPool{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.WorkerPool: %w", err)
	}

	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllWorkerpools, c.ExcludeWorkerpools, c.ExcludeWorkerpoolsRegex, c.ExcludeWorkerpoolsExcept) {
		return nil
	}

	zap.L().Info("Worker Pool: " + resource.Id + " " + resource.Name)
	return c.toHcl(resource, true, false, stateless, dependencies)
}

func (c WorkerPoolConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	pool := octopus.WorkerPool{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &pool)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.WorkerPool: %w", err)
	}

	if c.Excluder.IsResourceExcludedWithRegex(pool.Name, c.ExcludeAllWorkerpools, c.ExcludeWorkerpools, c.ExcludeWorkerpoolsRegex, c.ExcludeWorkerpoolsExcept) {
		return nil
	}

	return c.toHcl(pool, false, true, false, dependencies)
}

func (c WorkerPoolConverter) buildData(resourceName string, resourceDisplayName string) terraform.TerraformWorkerPoolData {
	return terraform.TerraformWorkerPoolData{
		Type:         octopusdeployWorkerPoolsDataType,
		Name:         resourceName,
		ResourceName: nil,
		Ids:          nil,
		PartialName:  strutil.StrPointer(resourceDisplayName),
		Skip:         0,
		Take:         1,
	}
}

// writeData appends the data block for stateless modules
func (c WorkerPoolConverter) writeData(file *hclwrite.File, resource octopus.WorkerPool, resourceName string) {
	terraformResource := c.buildData(resourceName, resource.Name)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

// toBashImport creates a bash script to import the resource
func (c WorkerPoolConverter) toBashImport(resourceType string, resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/WorkerPools" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No worker pool found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing worker pool ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, resourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c WorkerPoolConverter) toPowershellImport(resourceType string, resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/WorkerPools?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No worker pool found with the name $ResourceName"
	exit 1
}

echo "Importing worker pool $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, resourceType, resourceName), nil
		},
	})
}

func (c WorkerPoolConverter) toHcl(pool octopus.WorkerPool, _ bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.Excluder.IsResourceExcludedWithRegex(pool.Name, c.ExcludeAllWorkerpools, c.ExcludeWorkerpools, c.ExcludeWorkerpoolsRegex, c.ExcludeWorkerpoolsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + pool.Id)
		return nil
	}

	resourceName := "workerpool_" + sanitizer.SanitizeName(pool.Name)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = pool.Id
	thisResource.Name = pool.Name
	thisResource.ResourceType = c.GetResourceType()

	if pool.WorkerPoolType == "DynamicWorkerPool" {
		forceLookup := lookup || pool.Name == "Hosted Windows" || pool.Name == "Hosted Ubuntu"
		fallback := "Default Worker Pool"

		if forceLookup {
			c.createDynamicWorkerPoolLookupResource(resourceName,
				"workerpool_"+sanitizer.SanitizeName(fallback),
				&thisResource,
				pool,
				stateless)

			dependencies.AddResource(*c.createStandAloneLookupResource(
				"workerpool_"+sanitizer.SanitizeName(fallback),
				fallback))

		} else {
			if c.GenerateImportScripts {
				c.toBashImport(octopusdeployDynamicWorkerPoolResourceType, resourceName, pool.Name, dependencies)
				c.toPowershellImport(octopusdeployDynamicWorkerPoolResourceType, resourceName, pool.Name, dependencies)
			}
			c.createDynamicWorkerPoolResource(resourceName, &thisResource, dependencies, pool, stateless)
		}
	} else if pool.WorkerPoolType == "StaticWorkerPool" {
		forceLookup := lookup || pool.Name == "Default Worker Pool"
		fallback := "Hosted Ubuntu"
		fallback2 := "Hosted Windows"

		if forceLookup {
			c.createStaticWorkerPoolLookupResource(resourceName,
				"workerpool_"+sanitizer.SanitizeName(fallback),
				&thisResource,
				pool,
				stateless)

			dependencies.AddResource(*c.createStandAloneLookupResource(
				"workerpool_"+sanitizer.SanitizeName(fallback),
				fallback))

			dependencies.AddResource(*c.createStandAloneLookupResource(
				"workerpool_"+sanitizer.SanitizeName(fallback2),
				fallback2))

		} else {
			if c.GenerateImportScripts {
				c.toBashImport(octopusdeployStaticWorkerPoolResourcePool, resourceName, pool.Name, dependencies)
				c.toPowershellImport(octopusdeployStaticWorkerPoolResourcePool, resourceName, pool.Name, dependencies)
			}
			c.createStaticWorkerPoolResource(resourceName, &thisResource, pool, stateless)
		}
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c WorkerPoolConverter) GetResourceType() string {
	return "WorkerPools"
}

func (c WorkerPoolConverter) createDynamicWorkerPoolResource(resourceName string, thisResource *data.ResourceDetails, dependencies *data.ResourceDetailsCollection, pool octopus.WorkerPool, stateless bool) {
	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools) != 0 " +
			"? data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools[0].id " +
			": " + octopusdeployDynamicWorkerPoolResourceType + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployDynamicWorkerPoolResourceType + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployDynamicWorkerPoolResourceType + "." + resourceName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformWorkerPool{
			Type:         octopusdeployDynamicWorkerPoolResourceType,
			Name:         resourceName,
			Id:           strutil.InputPointerIfEnabled(c.IncludeIds, &pool.Id),
			SpaceId:      strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", pool.SpaceId)),
			ResourceName: pool.Name,
			Description:  strutil.TrimPointer(pool.Description),
			IsDefault:    pool.IsDefault,
			//SortOrder:    pool.SortOrder,
			WorkerType: pool.WorkerType,
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, pool, resourceName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools) != 0 ? 0 : 1}")
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
}

func (c WorkerPoolConverter) createStaticWorkerPoolResource(resourceName string, thisResource *data.ResourceDetails, pool octopus.WorkerPool, stateless bool) {
	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools) != 0 " +
			"? data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools[0].id " +
			": " + octopusdeployStaticWorkerPoolResourcePool + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployStaticWorkerPoolResourcePool + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployStaticWorkerPoolResourcePool + "." + resourceName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformWorkerPool{
			Type:         octopusdeployStaticWorkerPoolResourcePool,
			Name:         resourceName,
			ResourceName: pool.Name,
			Description:  strutil.TrimPointer(pool.Description),
			IsDefault:    pool.IsDefault,
			//SortOrder:    pool.SortOrder,
			WorkerType: pool.WorkerType,
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, pool, resourceName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools) != 0 ? 0 : 1}")
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
}

// createStandAloneLookupResource creates a data resource for the equivalent worker pool in an on-premise or cloud instance.
// This resource is only used by stateless modules. It allows a stateless module to be created on a cloud instance and applied
// to an on-premise instance, or vice versa.
func (c WorkerPoolConverter) createStandAloneLookupResource(resourceName string, resourceDisplayName string) *data.ResourceDetails {

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Name = resourceDisplayName
	// There is no id for this resource, so we use the name
	thisResource.Id = resourceName
	// This is not a real resource, so we use a mock resource type
	thisResource.ResourceType = "FallbackWorkerPool"

	thisResource.ToHcl = func() (string, error) {
		data := c.buildData(resourceName, resourceDisplayName)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(data, "data")
		file.Body().AppendBlock(block)
		return string(file.Bytes()), nil
	}

	return &thisResource
}

func (c WorkerPoolConverter) createStaticWorkerPoolLookupResource(resourceName string, fallbackResourceName string, thisResource *data.ResourceDetails, pool octopus.WorkerPool, stateless bool) {
	if stateless {
		// Stateless modules try to use the static worker pool first, and if that fails, use the dynamic worker pool.
		// This allows modules created from an on-premise instance to be used in a cloud instance.
		thisResource.Lookup = "${length(data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools) != 0 " +
			"? data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools[0].id " +
			": data." + octopusdeployWorkerPoolsDataType + "." + fallbackResourceName + ".worker_pools[0].id}"
	} else {
		thisResource.Lookup = "${data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools[0].id}"
	}

	thisResource.ToHcl = func() (string, error) {
		data := c.buildData(resourceName, pool.Name)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(data, "data")
		if !stateless {
			// Stateless modules may be used on cloud or on-premise deployments
			// We don't want to force the user to have a worker pool in the space
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a worker pool called \""+pool.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.worker_pools) != 0")
		}
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
}

func (c WorkerPoolConverter) createDynamicWorkerPoolLookupResource(resourceName string, fallbackResourceName string, thisResource *data.ResourceDetails, pool octopus.WorkerPool, stateless bool) {
	if stateless {
		// Stateless modules try to use the dynamic worker pool first, and if that fails, use the static worker pool
		// This allows a module created on a cloud instance to be used in an on-premise instance.
		thisResource.Lookup = "${length(data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools) != 0 " +
			"? data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools[0].id " +
			": data." + octopusdeployWorkerPoolsDataType + "." + fallbackResourceName + ".worker_pools[0].id}"
	} else {
		thisResource.Lookup = "${data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools[0].id}"
	}

	thisResource.ToHcl = func() (string, error) {
		data := c.buildData(resourceName, pool.Name)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(data, "data")
		if !stateless {
			// Stateless modules may be used on cloud or on-premise deployments
			// We don't want to force the user to have a worker pool in the space
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a worker pool called \""+pool.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.worker_pools) != 0")
		}
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
}
