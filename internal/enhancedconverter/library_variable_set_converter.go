package enhancedconverter

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
	"strings"
)

type LibraryVariableSetConverter struct {
	Client client.OctopusClient
}

func (c LibraryVariableSetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
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

func (c LibraryVariableSetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	resource := octopus.LibraryVariableSet{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, true, dependencies)
}

func (c LibraryVariableSetConverter) toHcl(resource octopus.LibraryVariableSet, recursive bool, dependencies *ResourceDetailsCollection) error {
	thisResource := ResourceDetails{}

	resourceName := "library_variable_set_" + util.SanitizeName(resource.Name)

	projectTemplates, projectTemplateMap := c.convertTemplates(resource.Templates, resourceName)

	if recursive {
		// The project group is a dependency that we need to lookup
		if util.EmptyIfNil(resource.ContentType) == "Variables" {
			err := VariableSetConverter{
				Client: c.Client,
			}.ToHclById(resource.VariableSetId, true, resourceName, "${octopusdeploy_library_variable_set."+resourceName+".id}", dependencies)

			if err != nil {
				return err
			}
		}

		// The templates are dependencies that we export as part of the project
		dependencies.AddResource(projectTemplateMap...)
	}

	thisResource.FileName = "space_population/library_variable_set_" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_library_variable_set." + resourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		if util.EmptyIfNil(resource.ContentType) == "Variables" {
			terraformResource := terraform.TerraformLibraryVariableSet{
				Type:         "octopusdeploy_library_variable_set",
				Name:         resourceName,
				ResourceName: resource.Name,
				Description:  resource.Description,
				Template:     projectTemplates,
			}

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		} else if util.EmptyIfNil(resource.ContentType) == "ScriptModule" {
			variable := octopus.VariableSet{}
			err := c.Client.GetResourceById("Variables", resource.VariableSetId, &variable)

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
				Type:         "octopusdeploy_script_module",
				Name:         resourceName,
				ResourceName: resource.Name,
				Description:  resource.Description,
				Script: terraform.TerraformScriptModuleScript{
					Body:   script,
					Syntax: scriptLanguage,
				},
			}

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
			return string(file.Bytes()), nil
		}

		return "", nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c LibraryVariableSetConverter) GetResourceType() string {
	return "LibraryVariableSets"
}

func (c LibraryVariableSetConverter) convertTemplates(actionPackages []octopus.Template, libraryName string) ([]terraform.TerraformTemplate, []ResourceDetails) {
	templateMap := make([]ResourceDetails, 0)
	collection := make([]terraform.TerraformTemplate, 0)
	for i, v := range actionPackages {
		collection = append(collection, terraform.TerraformTemplate{
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
