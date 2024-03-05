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
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
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
	Excluded                                args.ExcludeLibraryVariableSets
	ExcludeLibraryVariableSetsRegex         args.ExcludeLibraryVariableSets
	ExcludeLibraryVariableSetsExcept        args.ExcludeLibraryVariableSets
	ExcludeAllLibraryVariableSets           bool
	excludeLibraryVariableSetsRegexCompiled []*regexp.Regexp
	DummySecretVariableValues               bool
	DummySecretGenerator                    DummySecretGenerator
	Excluder                                ExcludeByName
	ErrGroup                                *errgroup.Group
}

func (c *LibraryVariableSetConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c *LibraryVariableSetConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c *LibraryVariableSetConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllLibraryVariableSets, c.Excluded, c.ExcludeLibraryVariableSetsRegex, c.ExcludeLibraryVariableSetsExcept) {
			continue
		}

		zap.L().Info("Library Variable Set: " + resource.Id)
		err = c.toHcl(resource, false, stateless, dependencies)

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
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

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
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

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

func (c *LibraryVariableSetConverter) toHcl(resource octopus.LibraryVariableSet, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded runbooks
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllLibraryVariableSets, c.Excluded, c.ExcludeLibraryVariableSetsRegex, c.ExcludeLibraryVariableSetsExcept) {
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

	block := gohcl.EncodeAsBlock(terraformResource, "resource")

	if stateless {
		hcl.WriteLifecyclePreventDestroyAttribute(block)
	}

	file.Body().AppendBlock(block)

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
			DefaultValue:    v.GetDefaultValueString(),
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
