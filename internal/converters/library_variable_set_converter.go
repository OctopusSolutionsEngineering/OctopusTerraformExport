package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type LibraryVariableSetConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	AccountsMap       map[string]string
}

func (c LibraryVariableSetConverter) ToHcl() (map[string]string, map[string]string, error) {
	resource := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &resource)

	if err != nil {
		return nil, nil, err
	}

	resources := map[string]string{}
	resourcesMap := map[string]string{}

	for _, v := range resource.Items {
		file := hclwrite.NewEmptyFile()

		resourceName := "library_variable_set_" + util.SanitizeName(v.Name)
		resourceIdProperty := "${octopusdeploy_library_variable_set." + resourceName + ".id}"

		terraformResource := terraform.TerraformLibraryVariableSet{
			Type:         "octopusdeploy_library_variable_set",
			Name:         resourceName,
			ResourceName: v.Name,
			Description:  v.Description,
			Template:     c.convertTemplate(v.Templates),
		}

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		resources["space_population/"+resourceName+".tf"] = string(file.Bytes())
		resourcesMap[v.Id] = resourceIdProperty

		// Export variable set
		variableSet, err := VariableSetConverter{
			Client:      c.Client,
			AccountsMap: c.AccountsMap,
		}.ToHclById(v.VariableSetId, util.SanitizeName(v.Name), resourceIdProperty)

		if err != nil {
			return nil, nil, err
		}

		// merge the maps
		for k, v := range variableSet {
			resources[k] = v
		}

	}

	return resources, resourcesMap, nil
}

func (c LibraryVariableSetConverter) ToHclById(id string) (map[string]string, map[string]string, error) {
	resource := octopus.LibraryVariableSet{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil, nil, err
	}

	file := hclwrite.NewEmptyFile()

	resourceName := "library_variable_set_" + util.SanitizeName(resource.Name)

	terraformResource := terraform.TerraformLibraryVariableSet{
		Type:         "octopusdeploy_library_variable_set",
		Name:         resourceName,
		ResourceName: resource.Name,
		Description:  resource.Description,
		Template:     c.convertTemplate(resource.Templates),
	}

	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

	return map[string]string{
			resourceName + ".tf": string(file.Bytes()),
		}, map[string]string{
			resource.Id: "${octopusdeploy_library_variable_set." + resourceName + ".id}",
		}, nil
}

func (c LibraryVariableSetConverter) GetResourceType() string {
	return "LibraryVariableSets"
}

func (c LibraryVariableSetConverter) convertTemplate(template []octopus.Template) []terraform.TerraformTemplate {
	terraformTemplates := make([]terraform.TerraformTemplate, 0)
	for _, v := range template {
		terraformTemplates = append(terraformTemplates, terraform.TerraformTemplate{
			Name:            v.Name,
			Label:           v.Label,
			HelpText:        v.HelpText,
			DefaultValue:    v.DefaultValue,
			DisplaySettings: v.DisplaySettings,
		})
	}

	return terraformTemplates
}
