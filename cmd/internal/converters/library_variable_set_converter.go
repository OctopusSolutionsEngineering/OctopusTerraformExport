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
	"regexp"
	"strings"
)

const octopusdeployLibraryVariableSetsDataType = "octopusdeploy_library_variable_sets"
const octopusdeployLibraryVariableSetsResourceType = "octopusdeploy_library_variable_set"
const octopusdeployScriptModuleResourceType = "octopusdeploy_script_module"

type LibraryVariableSetConverter struct {
	Client                                  client.OctopusClient
	VariableSetConverter                    ConverterByIdWithNameAndParent
	Excluded                                args.StringSliceArgs
	ExcludeLibraryVariableSetsRegex         args.StringSliceArgs
	ExcludeLibraryVariableSetsExcept        args.StringSliceArgs
	ExcludeAllLibraryVariableSets           bool
	excludeLibraryVariableSetsRegexCompiled []*regexp.Regexp
	DummySecretVariableValues               bool
	DummySecretGenerator                    DummySecretGenerator
	Excluder                                ExcludeByName
	ErrGroup                                *errgroup.Group
	LimitResourceCount                      int
	GenerateImportScripts                   bool
}

func (c *LibraryVariableSetConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c *LibraryVariableSetConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c *LibraryVariableSetConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllLibraryVariableSets {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.LibraryVariableSet]{
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
		if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllLibraryVariableSets, c.Excluded, c.ExcludeLibraryVariableSetsRegex, c.ExcludeLibraryVariableSetsExcept) {
			continue
		}

		zap.L().Info("Library Variable Set: " + resource.Id)
		err := c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *LibraryVariableSetConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c *LibraryVariableSetConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c *LibraryVariableSetConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.LibraryVariableSet{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Library Variable Set: " + resource.Id)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c *LibraryVariableSetConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.LibraryVariableSet{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Ignore excluded runbooks
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllLibraryVariableSets, c.Excluded, c.ExcludeLibraryVariableSetsRegex, c.ExcludeLibraryVariableSetsExcept) {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceName := "library_variable_set_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.Name = resource.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(resourceName, resource)

		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a library variable set called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.library_variable_sets) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	// All the variable templates need to be exported so they can be referenced individually.
	// There is no scenario where we use these lookups now. Either the space is exported, and the library variable set templates
	// are exported by getTemplateLookup(). Or we export a project with lookups, in which case tenant variables are not exported.
	// However, we may use this in the future, so we will keep it for now.
	for _, template := range resource.Templates {
		if template.Name == nil {
			continue
		}

		templateResource := data.ResourceDetails{}
		templateResource.Id = template.Id
		templateResource.Name = strutil.EmptyIfNil(template.Name)
		templateResource.ResourceType = "CommonTemplateMap"
		templateResource.Lookup = "tolist([for tmp in data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets[0].id.template : tmp.id if tmp.name == \"" + strutil.EmptyIfNil(template.Name) + "\"])[0]"
		dependencies.AddResource(templateResource)
	}

	return nil
}

func (c *LibraryVariableSetConverter) buildData(resourceName string, resource octopus.LibraryVariableSet) terraform.TerraformLibraryVariableSetData {
	return terraform.TerraformLibraryVariableSetData{
		Type:        octopusdeployLibraryVariableSetsDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c *LibraryVariableSetConverter) writeData(file *hclwrite.File, resource octopus.LibraryVariableSet, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

// toBashImport creates a bash script to import the resource
func (c LibraryVariableSetConverter) toBashImport(resourceType string, resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
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
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/LibraryVariableSets" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No target found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing target ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, resourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c LibraryVariableSetConverter) toPowershellImport(resourceType string, resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
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

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/LibraryVariableSets?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No library variable set found with the name $ResourceName"
	exit 1
}

echo "Importing library variable set $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, resourceType, resourceName), nil
		},
	})
}

func (c *LibraryVariableSetConverter) toHcl(resource octopus.LibraryVariableSet, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded runbooks
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllLibraryVariableSets, c.Excluded, c.ExcludeLibraryVariableSetsRegex, c.ExcludeLibraryVariableSetsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + resource.Id)
		return nil
	}

	thisResource := data.ResourceDetails{}

	// embedding the type allows files to be distinguished by script module and variable
	resourceName := "library_variable_set_" + sanitizer.SanitizeName(strutil.EmptyIfNil(resource.ContentType)) +
		"_" + sanitizer.SanitizeName(resource.Name)

	// The templates are dependencies that we export as part of the project
	projectTemplates, projectTemplateMap := c.convertTemplates(resource.Templates, resourceName, stateless)
	dependencies.AddResource(projectTemplateMap...)

	// The variables are a dependency that we need to export regardless of whether recursive is set.
	// That said, the variables themselves should only recursively export dependencies if we are
	// exporting a single project.
	if strutil.EmptyIfNil(resource.ContentType) == "Variables" {
		if stateless {
			err := c.VariableSetConverter.ToHclStatelessByIdAndName(
				resource.VariableSetId,
				recursive,
				strutil.EmptyIfNil(resource.ContentType)+" "+resource.Name,
				c.getParentLookup(stateless, resourceName),
				c.getParentCount(stateless, resourceName),
				dependencies)

			if err != nil {
				return err
			}
		} else {

			err := c.VariableSetConverter.ToHclByIdAndName(
				resource.VariableSetId,
				recursive,
				strutil.EmptyIfNil(resource.ContentType)+" "+resource.Name,
				c.getParentLookup(stateless, resourceName),
				c.getParentCount(stateless, resourceName),
				dependencies)

			if err != nil {
				return err
			}
		}

	}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.Name = resource.Name
	thisResource.ResourceType = c.GetResourceType()

	if strutil.EmptyIfNil(resource.ContentType) == "Variables" {
		thisResource.Lookup = c.getLibraryVariableSetLookup(stateless, resourceName)
		thisResource.Dependency = c.getLibraryVariableSetDependency(stateless, resourceName)
	} else if strutil.EmptyIfNil(resource.ContentType) == "ScriptModule" {
		thisResource.Lookup = c.getScriptModuleLookup(stateless, resourceName)
		thisResource.Dependency = c.getScriptModuleDependency(stateless, resourceName)
	}

	if strutil.EmptyIfNil(resource.ContentType) == "Variables" {

		if c.GenerateImportScripts {
			c.toBashImport(octopusdeployLibraryVariableSetsResourceType, resourceName, resource.Name, dependencies)
			c.toPowershellImport(octopusdeployLibraryVariableSetsResourceType, resourceName, resource.Name, dependencies)
		}
		thisResource.ToHcl = func() (string, error) {
			return c.writeLibraryVariableSet(resource, resourceName, projectTemplates, stateless)
		}
	} else if strutil.EmptyIfNil(resource.ContentType) == "ScriptModule" {
		if c.GenerateImportScripts {
			c.toBashImport(octopusdeployScriptModuleResourceType, resourceName, resource.Name, dependencies)
			c.toPowershellImport(octopusdeployScriptModuleResourceType, resourceName, resource.Name, dependencies)
		}
		thisResource.ToHcl = func() (string, error) {
			return c.writeScriptModule(resource, resourceName, stateless)
		}
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c *LibraryVariableSetConverter) writeLibraryVariableSet(resource octopus.LibraryVariableSet, resourceName string, projectTemplates []terraform.TerraformTemplate, stateless bool) (string, error) {
	terraformResource := terraform.TerraformLibraryVariableSet{
		Type:         octopusdeployLibraryVariableSetsResourceType,
		Name:         resourceName,
		ResourceName: resource.Name,
		Description:  resource.Description,
		Template:     projectTemplates,
	}

	file := hclwrite.NewEmptyFile()

	if stateless {
		c.writeData(file, resource, resourceName)
		terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets) != 0 ? 0 : 1}")
	}

	block := gohcl.EncodeAsBlock(terraformResource, "resource")

	if stateless {
		hcl.WriteLifecyclePreventDestroyAttribute(block)
	}

	file.Body().AppendBlock(block)

	return string(file.Bytes()), nil
}

func (c *LibraryVariableSetConverter) writeScriptModule(resource octopus.LibraryVariableSet, resourceName string, stateless bool) (string, error) {
	variable := octopus.VariableSet{}
	_, err := c.Client.GetSpaceResourceById("Variables", resource.VariableSetId, &variable)

	if err != nil {
		return "", err
	}

	script := ""
	scriptLanguage := ""
	for _, u := range variable.Variables {
		if u.Name == "Octopus.Script.Module["+resource.Name+"]" {
			script = strings.Clone(*u.Value)
		}

		if u.Name == "Octopus.Script.Module.Language["+resource.Name+"]" {
			scriptLanguage = strings.Clone(*u.Value)
		}
	}

	// An empty script language is assumed to be PowerShell
	if scriptLanguage == "" {
		scriptLanguage = "PowerShell"
	}

	terraformResource := terraform.TerraformScriptModule{
		Type:         octopusdeployScriptModuleResourceType,
		Name:         resourceName,
		ResourceName: resource.Name,
		Description:  resource.Description,
		Script: terraform.TerraformScriptModuleScript{
			Body:   script,
			Syntax: scriptLanguage,
		},
	}

	file := hclwrite.NewEmptyFile()

	if stateless {
		c.writeData(file, resource, resourceName)
		terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets) != 0 ? 0 : 1}")
	}

	block := gohcl.EncodeAsBlock(terraformResource, "resource")

	if stateless {
		hcl.WriteLifecyclePreventDestroyAttribute(block)
	}

	file.Body().AppendBlock(block)
	return string(file.Bytes()), nil
}

func (c *LibraryVariableSetConverter) GetResourceType() string {
	return "LibraryVariableSets"
}

func (c *LibraryVariableSetConverter) getParentLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets) != 0 ? " +
			"data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets[0].id : " +
			octopusdeployLibraryVariableSetsResourceType + "." + resourceName + "[0].id}"
	}

	return "${" + octopusdeployLibraryVariableSetsResourceType + "." + resourceName + ".id}"
}

func (c *LibraryVariableSetConverter) getParentCount(stateless bool, resourceName string) *string {
	if stateless {
		return strutil.StrPointer("${length(data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets) != 0 ? 0 : 1}")
	}

	return nil
}

func (c *LibraryVariableSetConverter) getLibraryVariableSetLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets) != 0 " +
			"? data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets[0].id " +
			": " + octopusdeployLibraryVariableSetsResourceType + "." + resourceName + "[0].id}"
	}

	return "${" + octopusdeployLibraryVariableSetsResourceType + "." + resourceName + ".id}"
}

func (c *LibraryVariableSetConverter) getLibraryVariableSetDependency(stateless bool, resourceName string) string {
	if stateless {
		return "${" + octopusdeployLibraryVariableSetsResourceType + "." + resourceName + "}"
	}

	return ""
}

func (c *LibraryVariableSetConverter) getScriptModuleLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets) != 0 " +
			"? data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets[0].id " +
			": " + octopusdeployScriptModuleResourceType + "." + resourceName + "[0].id}"
	}

	return "${" + octopusdeployScriptModuleResourceType + "." + resourceName + ".id}"
}

func (c *LibraryVariableSetConverter) getScriptModuleDependency(stateless bool, resourceName string) string {
	if stateless {
		return "${" + octopusdeployScriptModuleResourceType + "." + resourceName + "}"
	}

	return ""
}

func (c *LibraryVariableSetConverter) convertTemplates(actionPackages []octopus.Template, libraryName string, stateless bool) ([]terraform.TerraformTemplate, []data.ResourceDetails) {
	templateMap := make([]data.ResourceDetails, 0)
	collection := make([]terraform.TerraformTemplate, 0)
	for i, v := range actionPackages {
		collection = append(collection, terraform.TerraformTemplate{
			Name:            v.Name,
			Label:           v.Label,
			HelpText:        v.HelpText,
			DefaultValue:    strutil.EscapeDollarCurlyPointer(v.GetDefaultValueString()),
			DisplaySettings: v.DisplaySettings,
		})

		templateMap = append(templateMap, data.ResourceDetails{
			Id:           v.Id,
			ResourceType: "CommonTemplateMap",
			Lookup:       c.getTemplateLookup(stateless, libraryName, i),
			FileName:     "",
			ToHcl:        nil,
		})
	}
	return collection, templateMap
}

func (c *LibraryVariableSetConverter) getTemplateLookup(stateless bool, libraryName string, index int) string {
	if stateless {
		return "${length(data." + octopusdeployLibraryVariableSetsDataType + "." + libraryName + ".library_variable_sets) != 0 " +
			"? data." + octopusdeployLibraryVariableSetsDataType + "." + libraryName + ".library_variable_sets[0].template[" + fmt.Sprint(index) + "] " +
			": " + octopusdeployLibraryVariableSetsResourceType + "." + libraryName + "[0].template[" + fmt.Sprint(index) + "].id}"
	}

	return "${" + octopusdeployLibraryVariableSetsResourceType + "." + libraryName + ".template[" + fmt.Sprint(index) + "].id}"
}
