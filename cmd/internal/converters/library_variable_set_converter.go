package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/args"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
	"regexp"
	"strings"
)

type LibraryVariableSetConverter struct {
	Client                                  client.OctopusClient
	VariableSetConverter                    ConverterByIdWithNameAndParent
	Excluded                                args.ExcludeLibraryVariableSets
	ExcludeLibraryVariableSetsRegex         args.ExcludeLibraryVariableSets
	excludeLibraryVariableSetsRegexCompiled []*regexp.Regexp
}

func (c *LibraryVariableSetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.LibraryVariableSet]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		err = c.toHcl(resource, false, dependencies)

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

	resource := octopus2.LibraryVariableSet{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, true, dependencies)
}

func (c *LibraryVariableSetConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.LibraryVariableSet{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if c.Excluded != nil && slices.Index(c.Excluded, resource.Name) != -1 {
		return nil
	}

	thisResource := ResourceDetails{}

	resourceName := "library_variable_set_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data.octopusdeploy_library_variable_sets." + resourceName + ".library_variable_sets[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform2.TerraformLibraryVariableSetData{
			Type:        "octopusdeploy_library_variable_sets",
			Name:        resourceName,
			Ids:         nil,
			PartialName: resource.Name,
			Skip:        0,
			Take:        1,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c *LibraryVariableSetConverter) toHcl(resource octopus2.LibraryVariableSet, recursive bool, dependencies *ResourceDetailsCollection) error {
	c.compileRegexes()

	if c.libraryVariableSetIsExcluded(resource) {
		return nil
	}

	thisResource := ResourceDetails{}

	resourceName := "library_variable_set_" + sanitizer.SanitizeName(resource.Name)

	// The templates are dependencies that we export as part of the project
	projectTemplates, projectTemplateMap := c.convertTemplates(resource.Templates, resourceName)
	dependencies.AddResource(projectTemplateMap...)

	// The project group is a dependency that we need to lookup regardless of whether recursive is set
	if strutil.EmptyIfNil(resource.ContentType) == "Variables" {
		err := c.VariableSetConverter.ToHclByIdAndName(resource.VariableSetId, resourceName, "${octopusdeploy_library_variable_set."+resourceName+".id}", dependencies)

		if err != nil {
			return err
		}
	}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_library_variable_set." + resourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		if strutil.EmptyIfNil(resource.ContentType) == "Variables" {
			terraformResource := terraform2.TerraformLibraryVariableSet{
				Type:         "octopusdeploy_library_variable_set",
				Name:         resourceName,
				ResourceName: resource.Name,
				Description:  resource.Description,
				Template:     projectTemplates,
			}

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_library_variable_set." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			// Add a data lookup to allow projects to quickly switch to using existing environments
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# To use an existing environment, delete the resource above and use the following lookup instead:\n" +
					"# data.octopusdeploy_library_variable_sets." + resourceName + ".library_variable_sets[0].id\n"),
				SpacesBefore: 0,
			}})
			terraformDataResource := terraform2.TerraformLibraryVariableSetData{
				Type:        "octopusdeploy_library_variable_sets",
				Name:        resourceName,
				Ids:         nil,
				PartialName: resource.Name,
				Skip:        0,
				Take:        1,
			}
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformDataResource, "data"))

			return string(file.Bytes()), nil
		} else if strutil.EmptyIfNil(resource.ContentType) == "ScriptModule" {
			variable := octopus2.VariableSet{}
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

			terraformResource := terraform2.TerraformScriptModule{
				Type:         "octopusdeploy_script_module",
				Name:         resourceName,
				ResourceName: resource.Name,
				Description:  resource.Description,
				Script: terraform2.TerraformScriptModuleScript{
					Body:   script,
					Syntax: scriptLanguage,
				},
			}

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_script_module." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
			return string(file.Bytes()), nil
		}

		return "", nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c *LibraryVariableSetConverter) GetResourceType() string {
	return "LibraryVariableSets"
}

func (c *LibraryVariableSetConverter) convertTemplates(actionPackages []octopus2.Template, libraryName string) ([]terraform2.TerraformTemplate, []ResourceDetails) {
	templateMap := make([]ResourceDetails, 0)
	collection := make([]terraform2.TerraformTemplate, 0)
	for i, v := range actionPackages {
		collection = append(collection, terraform2.TerraformTemplate{
			Name:            v.Name,
			Label:           v.Label,
			HelpText:        v.HelpText,
			DefaultValue:    v.DefaultValue,
			DisplaySettings: v.DisplaySettings,
		})

		templateMap = append(templateMap, ResourceDetails{
			Id:           v.Id,
			ResourceType: "CommonTemplateMap",
			Lookup:       "${octopusdeploy_library_variable_set." + libraryName + ".template[" + fmt.Sprint(i) + "].id}",
			FileName:     "",
			ToHcl:        nil,
		})
	}
	return collection, templateMap
}

func (c *LibraryVariableSetConverter) compileRegexes() {
	if c.ExcludeLibraryVariableSetsRegex != nil {
		c.excludeLibraryVariableSetsRegexCompiled = lo.FilterMap(c.ExcludeLibraryVariableSetsRegex, func(x string, index int) (*regexp.Regexp, bool) {
			re, err := regexp.Compile(x)
			if err != nil {
				return nil, false
			}
			return re, true
		})
	}
}

func (c *LibraryVariableSetConverter) libraryVariableSetIsExcluded(resource octopus2.LibraryVariableSet) bool {
	if c.Excluded != nil && slices.Index(c.Excluded, resource.Name) != -1 {
		return true
	}

	if c.excludeLibraryVariableSetsRegexCompiled != nil {
		return lo.SomeBy(c.excludeLibraryVariableSetsRegexCompiled, func(x *regexp.Regexp) bool {
			return x.MatchString(resource.Name)
		})
	}

	return false
}
