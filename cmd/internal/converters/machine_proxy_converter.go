package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/intutil"
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

const octopusdeployMachineProxyDataType = "octopusdeploy_machine_proxies"
const octopusdeployMachineProxyResourceType = "octopusdeploy_machine_proxy"

type MachineProxyConverter struct {
	Client                      client.OctopusClient
	ErrGroup                    *errgroup.Group
	ExcludeMachineProxies       args.StringSliceArgs
	ExcludeMachineProxiesRegex  args.StringSliceArgs
	ExcludeMachineProxiesExcept args.StringSliceArgs
	ExcludeAllMachineProxies    bool
	Excluder                    ExcludeByName
	LimitResourceCount          int
	IncludeSpaceInPopulation    bool
	IncludeIds                  bool
	GenerateImportScripts       bool
	DummySecretVariableValues   bool
	DummySecretGenerator        dummy.DummySecretGenerator
}

func (c MachineProxyConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c MachineProxyConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c MachineProxyConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllMachineProxies {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.MachineProxy]{
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

		if c.Excluder.IsResourceExcludedWithRegex(
			resource.Name,
			c.ExcludeAllMachineProxies,
			c.ExcludeMachineProxies,
			c.ExcludeMachineProxiesRegex,
			c.ExcludeMachineProxiesExcept) {
			continue
		}

		zap.L().Info("Machine proxy: " + resource.Id + " " + resource.Name)
		err := c.toHcl(resource, false, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c MachineProxyConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c MachineProxyConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c MachineProxyConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {

	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.MachineProxy{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.MachineProxy: %w", err)
	}

	if c.Excluder.IsResourceExcludedWithRegex(
		resource.Name,
		c.ExcludeAllMachineProxies,
		c.ExcludeMachineProxies,
		c.ExcludeMachineProxiesRegex,
		c.ExcludeMachineProxiesExcept) {
		return nil
	}

	zap.L().Info("Machine proxy: " + resource.Id + " " + resource.Name)
	return c.toHcl(resource, false, false, stateless, dependencies)
}

func (c MachineProxyConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.MachineProxy{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.MachineProxy: %w", err)
	}

	if c.Excluder.IsResourceExcludedWithRegex(resource.Name,
		c.ExcludeAllMachineProxies,
		c.ExcludeMachineProxies,
		c.ExcludeMachineProxiesRegex,
		c.ExcludeMachineProxiesExcept) {
		return nil
	}

	return c.toHcl(resource, false, true, false, dependencies)
}

func (c MachineProxyConverter) buildData(resourceName string, name string) terraform.TerraformProjectGroupData {
	return terraform.TerraformProjectGroupData{
		Type:        octopusdeployMachineProxyDataType,
		Name:        name,
		Ids:         nil,
		PartialName: resourceName,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c MachineProxyConverter) writeData(file *hclwrite.File, name string, resourceName string) {
	terraformResource := c.buildData(resourceName, name)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

// toBashImport creates a bash script to import the resource
func (c MachineProxyConverter) toBashImport(resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Proxies" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No machine proxy found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing machine proxy ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, octopusdeployMachineProxyResourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c MachineProxyConverter) toPowershellImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Proxies?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No machine proxy found with the name $ResourceName"
	exit 1
}

echo "Importing machine proxy $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, octopusdeployMachineProxyResourceType, resourceName), nil
		},
	})
}

func (c MachineProxyConverter) toHcl(resource octopus.MachineProxy, recursive bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.Excluder.IsResourceExcludedWithRegex(
		resource.Name,
		c.ExcludeAllMachineProxies,
		c.ExcludeMachineProxies,
		c.ExcludeMachineProxiesRegex,
		c.ExcludeMachineProxiesExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + resource.Id)
		return nil
	}

	thisResource := data.ResourceDetails{}

	machineProxyName := "machine_proxy_" + sanitizer.SanitizeName(resource.Name)

	if c.GenerateImportScripts {
		c.toBashImport(machineProxyName, resource.Name, dependencies)
		c.toPowershellImport(machineProxyName, resource.Name, dependencies)
	}

	thisResource.FileName = "space_population/machine_proxy_" + machineProxyName + ".tf"
	thisResource.Id = resource.Id
	thisResource.Name = resource.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Parameters = []data.ResourceParameter{
		{
			Label:         "Machine proxy " + resource.Name + " password",
			Description:   "The password associated with the machine proxy \"" + resource.Name + "\"",
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, resource.Name, "Password"),
			ParameterType: "Password",
			Sensitive:     true,
			VariableName:  machineProxyName,
		},
	}

	if lookup {
		thisResource.Lookup = "${data." + octopusdeployMachineProxyDataType + "." + machineProxyName + ".machine_proxies[0].id}"
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData("${var."+machineProxyName+"_name}", machineProxyName)
			file := hclwrite.NewEmptyFile()
			c.writeMachineProxyNameVariable(file, machineProxyName, resource.Name)
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a machine proxy called ${var."+machineProxyName+"_name}. This resource must exist in the space before this Terraform configuration is applied.", "length(self.machine_proxies) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	} else {
		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployMachineProxyDataType + "." + machineProxyName + ".machine_proxies) != 0 " +
				"? data." + octopusdeployMachineProxyDataType + "." + machineProxyName + ".machine_proxies[0].id " +
				": " + octopusdeployMachineProxyResourceType + "." + machineProxyName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployMachineProxyResourceType + "." + machineProxyName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeployMachineProxyResourceType + "." + machineProxyName + ".id}"
		}

		thisResource.ToHcl = func() (string, error) {
			passwordName := naming.MachineProxyPassword(resource)

			terraformResource := terraform.TerraformMachineProxy{
				Type:         octopusdeployMachineProxyResourceType,
				Name:         machineProxyName,
				Count:        nil,
				ResourceName: "${var." + machineProxyName + "_name}",
				Id:           strutil.InputPointerIfEnabled(c.IncludeIds, &resource.Id),
				SpaceId:      strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", resource.SpaceId)),
				Host:         resource.Host,
				Password:     "${var." + passwordName + "}",
				Username:     resource.Username,
				Port:         intutil.NilIfZero(resource.Port),
			}
			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, machineProxyName, "${var."+machineProxyName+"_name}")
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployMachineProxyDataType + "." + machineProxyName + ".machine_proxies) != 0 ? 0 : 1}")
			}

			c.writeMachineProxyNameVariable(file, machineProxyName, resource.Name)

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			if stateless {
				hcl.WriteLifecyclePreventDestroyAttribute(block)
			}

			file.Body().AppendBlock(block)

			secretVariableResource := terraform.TerraformVariable{
				Name:        passwordName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The secret variable value associated with the machine proxy \"" + resource.Name + "\"",
			}

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
				dependencies.AddDummy(data.DummyVariableReference{
					VariableName: passwordName,
					ResourceName: resource.Name,
					ResourceType: c.GetResourceType(),
				})
			}

			variableBlock := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(variableBlock, "type", "string")
			file.Body().AppendBlock(variableBlock)

			return string(file.Bytes()), nil
		}
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c MachineProxyConverter) writeMachineProxyNameVariable(file *hclwrite.File, proxyName string, machineGroupResourceName string) {
	machineProxyNameVariableResource := terraform.TerraformVariable{
		Name:        proxyName + "_name",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The name of the machine proxy to lookup",
		Default:     &machineGroupResourceName,
	}

	block := gohcl.EncodeAsBlock(machineProxyNameVariableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)
}

func (c MachineProxyConverter) GetResourceType() string {
	return "Proxies"
}
