package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"regexp"
	"strings"
)

const octopusdeployLibraryVariableSetsDataType = "octopusdeploy_library_variable_sets"
const octopusdeployLibraryVariableSetsResourceType = "octopusdeploy_library_variable_set"
const octopusdeployScriptModuleResourceType = "octopusdeploy_script_module"

type LibraryVariableSetConverter struct {
	Client                                  client.OctopusClient
	VariableSetConverter                    ConverterByIdWithNameAndParent
	Excluded                                args.ExcludeLibraryVariableSets
	ExcludeLibraryVariableSetsRegex         args.ExcludeLibraryVariableSets
	ExcludeLibraryVariableSetsExcept        args.ExcludeLibraryVariableSets
	ExcludeAllLibraryVariableSets           bool
	excludeLibraryVariableSetsRegexCompiled []*regexp.Regexp
	DummySecretVariableValues               bool
	DummySecretGenerator                    DummySecretGenerator
	Excluder                                ExcludeByName
}

func (c *LibraryVariableSetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Library Variable Set: " + resource.Id)
		err = c.toHcl(resource, false, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *LibraryVariableSetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.LibraryVariableSet{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Library Variable Set: " + resource.Id)
	return c.toHcl(resource, true, false, dependencies)
}

func (c *LibraryVariableSetConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.LibraryVariableSet{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Ignore excluded runbooks
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllLibraryVariableSets, c.Excluded, c.ExcludeLibraryVariableSetsRegex, c.ExcludeLibraryVariableSetsExcept) {
		return nil
	}

	thisResource := ResourceDetails{}

	resourceName := "library_variable_set_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
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
	return nil
}

func (c LibraryVariableSetConverter) buildData(resourceName string, resource octopus.LibraryVariableSet) terraform.TerraformLibraryVariableSetData {
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
func (c LibraryVariableSetConverter) writeData(file *hclwrite.File, resource octopus.LibraryVariableSet, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c *LibraryVariableSetConverter) toHcl(resource octopus.LibraryVariableSet, _ bool, stateless bool, dependencies *ResourceDetailsCollection) error {
	// Ignore excluded runbooks
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllLibraryVariableSets, c.Excluded, c.ExcludeLibraryVariableSetsRegex, c.ExcludeLibraryVariableSetsExcept) {
		return nil
	}

	thisResource := ResourceDetails{}

	// embedding the type allows files to be distinguished by script module and variable
	resourceName := "library_variable_set_" + sanitizer.SanitizeName(strutil.EmptyIfNil(resource.ContentType)) +
		"_" + sanitizer.SanitizeName(resource.Name)

	// The templates are dependencies that we export as part of the project
	projectTemplates, projectTemplateMap := c.convertTemplates(resource.Templates, resourceName)
	dependencies.AddResource(projectTemplateMap...)

	// The project group is a dependency that we need to lookup regardless of whether recursive is set
	if strutil.EmptyIfNil(resource.ContentType) == "Variables" {
		var parentCount *string = nil
		if stateless {
			parentCount = strutil.StrPointer("${length(data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets) != 0 ? 0 : 1}")
		}

		err := c.VariableSetConverter.ToHclByIdAndName(
			resource.VariableSetId,
			resourceName,
			"${"+octopusdeployLibraryVariableSetsResourceType+"."+resourceName+".id}",
			parentCount,
			dependencies)

		if err != nil {
			return err
		}

	}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()

	if strutil.EmptyIfNil(resource.ContentType) == "Variables" {
		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets) != 0 " +
				"? data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets[0].id " +
				": " + octopusdeployLibraryVariableSetsResourceType + "." + resourceName + "[0].id}"
		} else {
			thisResource.Lookup = "${" + octopusdeployLibraryVariableSetsResourceType + "." + resourceName + ".id}"
		}

	} else if strutil.EmptyIfNil(resource.ContentType) == "ScriptModule" {
		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployScriptModuleResourceType + "." + resourceName + ".library_variable_sets) != 0 " +
				"? data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets[0].id " +
				": " + octopusdeployScriptModuleResourceType + "." + resourceName + "[0].id}"
		} else {
			thisResource.Lookup = "${" + octopusdeployScriptModuleResourceType + "." + resourceName + ".id}"
		}
	}

	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		if strutil.EmptyIfNil(resource.ContentType) == "Variables" {
			return c.writeLibraryVariableSet(resource, resourceName, projectTemplates, stateless, file)
		} else if strutil.EmptyIfNil(resource.ContentType) == "ScriptModule" {
			return c.writeScriptModule(resource, resourceName, stateless, file)
		}

		return "", nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c *LibraryVariableSetConverter) writeLibraryVariableSet(resource octopus.LibraryVariableSet, resourceName string, projectTemplates []terraform.TerraformTemplate, stateless bool, file *hclwrite.File) (string, error) {
	terraformResource := terraform.TerraformLibraryVariableSet{
		Type:         octopusdeployLibraryVariableSetsResourceType,
		Name:         resourceName,
		ResourceName: resource.Name,
		Description:  resource.Description,
		Template:     projectTemplates,
	}

	if stateless {
		c.writeData(file, resource, resourceName)
		terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets) != 0 ? 0 : 1}")
	}

	// Add a comment with the import command
	baseUrl, _ := c.Client.GetSpaceBaseUrl()
	file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), resource.Name, octopusdeployLibraryVariableSetsResourceType, resourceName))

	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

	// Add a data lookup to allow projects to quickly switch to using existing environments
	file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
		Type: hclsyntax.TokenComment,
		Bytes: []byte("# To use an existing environment, delete the resource above and use the following lookup instead:\n" +
			"# data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets[0].id\n"),
		SpacesBefore: 0,
	}})
	terraformDataResource := terraform.TerraformLibraryVariableSetData{
		Type:        octopusdeployLibraryVariableSetsDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: resource.Name,
		Skip:        0,
		Take:        1,
	}
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformDataResource, "data"))

	return string(file.Bytes()), nil
}

func (c *LibraryVariableSetConverter) writeScriptModule(resource octopus.LibraryVariableSet, resourceName string, stateless bool, file *hclwrite.File) (string, error) {
	variable := octopus.VariableSet{}
	_, err := c.Client.GetResourceById("Variables", resource.VariableSetId, &variable)

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

	if stateless {
		c.writeData(file, resource, resourceName)
		terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployLibraryVariableSetsDataType + "." + resourceName + ".library_variable_sets) != 0 ? 0 : 1}")
	}

	// Add a comment with the import command
	baseUrl, _ := c.Client.GetSpaceBaseUrl()
	file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
		Type: hclsyntax.TokenComment,
		Bytes: []byte("# Import existing resources with the following commands:\n" +
			"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
			"# terraform import " + octopusdeployScriptModuleResourceType + "." + resourceName + " ${RESOURCE_ID}\n"),
		SpacesBefore: 0,
	}})

	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
	return string(file.Bytes()), nil
}

func (c *LibraryVariableSetConverter) GetResourceType() string {
	return "LibraryVariableSets"
}

func (c *LibraryVariableSetConverter) convertTemplates(actionPackages []octopus.Template, libraryName string) ([]terraform.TerraformTemplate, []ResourceDetails) {
	templateMap := make([]ResourceDetails, 0)
	collection := make([]terraform.TerraformTemplate, 0)
	for i, v := range actionPackages {
		collection = append(collection, terraform.TerraformTemplate{
			Name:            v.Name,
			Label:           v.Label,
			HelpText:        v.HelpText,
			DefaultValue:    v.GetDefaultValueString(),
			DisplaySettings: v.DisplaySettings,
		})

		templateMap = append(templateMap, ResourceDetails{
			Id:           v.Id,
			ResourceType: "CommonTemplateMap",
			Lookup:       "${" + octopusdeployLibraryVariableSetsResourceType + "." + libraryName + ".template[" + fmt.Sprint(i) + "].id}",
			FileName:     "",
			ToHcl:        nil,
		})
	}
	return collection, templateMap
}
