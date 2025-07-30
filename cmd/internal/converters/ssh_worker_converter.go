package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/boolutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/intutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

const octopusdeploySshWorkerDataType = "octopusdeploy_workers"
const octopusdeploySshWorkerResourceType = "octopusdeploy_ssh_connection_worker"

type SshWorkerConverter struct {
	BaseWorkerConverter
	AccountConverter ConverterAndLookupWithStatelessById
}

func (c SshWorkerConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c SshWorkerConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c SshWorkerConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllWorkers {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.Worker]{
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

		if !c.isSshWorker(resource) {
			continue
		}

		zap.L().Info("SSH Worker: " + resource.Id + " " + resource.Name)
		err := c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c SshWorkerConverter) isSshWorker(resource octopus.Worker) bool {
	return resource.Endpoint.CommunicationStyle == "Ssh"
}

func (c SshWorkerConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c SshWorkerConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c SshWorkerConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Worker{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.KubernetesEndpointResource: %w", err)
	}

	if !c.isSshWorker(resource) {
		return nil
	}

	zap.L().Info("SSh Worker: " + resource.Id + " " + resource.Name)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c SshWorkerConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Worker{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.KubernetesEndpointResource: %w", err)
	}

	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllWorkers, c.ExcludeWorkers, c.ExcludeWorkersRegex, c.ExcludeWorkersExcept) {
		return nil
	}

	if !c.isSshWorker(resource) {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceName := "worker_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.Name = resource.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeploySshWorkerDataType + "." + resourceName + ".workers[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(resourceName, resource)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a worker called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.deployment_targets) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c SshWorkerConverter) buildData(resourceName string, resource octopus.Worker) terraform.TerraformWorkersData {
	return terraform.TerraformWorkersData{
		Type:        octopusdeploySshWorkerDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c SshWorkerConverter) writeData(file *hclwrite.File, resource octopus.Worker, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

// toBashImport creates a bash script to import the resource
func (c SshWorkerConverter) toBashImport(resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Workers" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No target found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing target ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, octopusdeploySshWorkerResourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c SshWorkerConverter) toPowershellImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Workers?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No target found with the name $ResourceName"
	exit 1
}

echo "Importing target $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, octopusdeploySshWorkerResourceType, resourceName), nil
		},
	})
}

func (c SshWorkerConverter) toHcl(worker octopus.Worker, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(worker.Name, c.ExcludeAllWorkers, c.ExcludeWorkers, c.ExcludeWorkersRegex, c.ExcludeWorkersExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + worker.Id)
		return nil
	}

	if !c.isSshWorker(worker) {
		return nil
	}

	if recursive {
		if stateless {
			if err := c.exportStatelessDependencies(worker, dependencies); err != nil {
				return err
			}
		} else {
			if err := c.exportDependencies(worker, dependencies); err != nil {
				return err
			}
		}
	}

	workerName := "worker_" + sanitizer.SanitizeName(worker.Name)

	if c.GenerateImportScripts && !stateless {
		c.toBashImport(workerName, worker.Name, dependencies)
		c.toPowershellImport(workerName, worker.Name, dependencies)
	}

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + workerName + ".tf"
	thisResource.Id = worker.Id
	thisResource.Name = worker.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = c.getLookup(stateless, workerName)
	thisResource.Dependency = c.getDependency(stateless, workerName)

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformSshWorker{
			Type:            octopusdeploySshWorkerResourceType,
			Name:            workerName,
			Count:           nil,
			Id:              strutil.InputPointerIfEnabled(c.IncludeIds, &worker.Id),
			SpaceId:         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", worker.SpaceId)),
			ResourceName:    worker.Name,
			AccountId:       dependencies.GetResource("Accounts", strutil.EmptyIfNil(worker.Endpoint.AccountId)),
			DotnetPlatform:  strutil.EmptyIfNil(worker.Endpoint.DotNetCorePlatform),
			Fingerprint:     strutil.EmptyIfNil(worker.Endpoint.Fingerprint),
			Host:            strutil.EmptyIfNil(worker.Endpoint.Host),
			Port:            intutil.ZeroIfNil(worker.Endpoint.Port),
			WorkerPoolIds:   dependencies.GetResources("WorkerPools", worker.WorkerPoolIds...),
			MachinePolicyId: c.getMachinePolicy(worker.MachinePolicyId, dependencies),
			ProxyId:         nil,
			IsDisabled:      boolutil.NilIfFalse(worker.IsDisabled),
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, worker, workerName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeploySshWorkerDataType + "." + workerName + ".deployment_targets) != 0 ? 0 : 1}")
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c SshWorkerConverter) GetResourceType() string {
	return "Workers"
}

func (c SshWorkerConverter) getMachinePolicy(machine string, dependencies *data.ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c SshWorkerConverter) getWorkerPool(pool *string, dependencies *data.ResourceDetailsCollection) *string {
	if len(strutil.EmptyIfNil(pool)) == 0 {
		return nil
	}

	workerPoolLookup := dependencies.GetResource("WorkerPools", *pool)
	if workerPoolLookup == "" {
		return nil
	}

	return &workerPoolLookup
}

func (c SshWorkerConverter) exportDependencies(target octopus.Worker, dependencies *data.ResourceDetailsCollection) error {
	// The machine policies need to be exported
	if err := c.MachinePolicyConverter.ToHclById(target.MachinePolicyId, dependencies); err != nil {
		return err
	}

	// Export the accounts
	if target.Endpoint.AccountId != nil {
		if err := c.AccountConverter.ToHclById(*target.Endpoint.AccountId, dependencies); err != nil {
			return err
		}
	}

	return nil
}

func (c SshWorkerConverter) exportStatelessDependencies(target octopus.Worker, dependencies *data.ResourceDetailsCollection) error {
	// The machine policies need to be exported
	if err := c.MachinePolicyConverter.ToHclStatelessById(target.MachinePolicyId, dependencies); err != nil {
		return err
	}

	// Export the accounts
	if target.Endpoint.AccountId != nil {
		if err := c.AccountConverter.ToHclStatelessById(*target.Endpoint.AccountId, dependencies); err != nil {
			return err
		}
	}

	return nil
}

func (c *SshWorkerConverter) getLookup(stateless bool, targetName string) string {
	if stateless {
		return "${length(data." + octopusdeploySshWorkerDataType + "." + targetName + ".workers) != 0 " +
			"? data." + octopusdeploySshWorkerDataType + "." + targetName + ".workers[0].id " +
			": " + octopusdeploySshWorkerResourceType + "." + targetName + "[0].id}"
	}
	return "${" + octopusdeploySshWorkerResourceType + "." + targetName + ".id}"
}

func (c *SshWorkerConverter) getDependency(stateless bool, targetName string) string {
	if stateless {
		return "${" + octopusdeploySshWorkerResourceType + "." + targetName + "}"
	}
	return ""
}
