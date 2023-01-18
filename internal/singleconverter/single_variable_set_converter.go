package singleconverter

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type SingleVariableSetConverter struct {
	Client client.OctopusClient
}

func (c SingleVariableSetConverter) ToHclById(id string, parentName string, parentId string, dependencies *ResourceDetailsCollection) error {
	resource := octopus.VariableSet{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	file := hclwrite.NewEmptyFile()

	for _, v := range resource.Variables {
		thisResource := ResourceDetails{}

		resourceName := parentName + "_" + util.SanitizeName(v.Name)

		thisResource.FileName = "space_population/project_variable_" + resourceName + ".tf"
		thisResource.Id = v.Id
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = "${octopusdeploy_variable." + resourceName + ".id}"
		thisResource.ToHcl = func(resources map[string]ResourceDetails) (string, error) {
			terraformResource := terraform.TerraformProjectVariable{
				Name:           resourceName,
				Type:           "octopusdeploy_variable",
				OwnerId:        dependencies.GetResource("LibraryVariableSets", parentId),
				Value:          dependencies.GetResourcePointer("Accounts", v.Value),
				ResourceName:   v.Name,
				ResourceType:   v.Type,
				Description:    v.Description,
				SensitiveValue: c.convertSecretValue(v, parentName),
				IsSensitive:    v.IsSensitive,
				Prompt:         c.convertPrompt(v.Prompt),
			}

			if v.IsSensitive {
				secretVariableResource := terraform.TerraformVariable{
					Name:        parentName,
					Type:        "string",
					Nullable:    false,
					Sensitive:   true,
					Description: "The secret variable value associated with the variable " + v.Name,
				}

				block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
				util.WriteUnquotedAttribute(block, "type", "string")
				file.Body().AppendBlock(block)
			}

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}
		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c SingleVariableSetConverter) GetResourceType() string {
	return "Variables"
}

func (c SingleVariableSetConverter) convertSecretValue(variable octopus.Variable, parentName string) *string {
	if variable.IsSensitive {
		value := "${var." + parentName + "}"
		return &value
	}

	return nil
}

func (c SingleVariableSetConverter) convertPrompt(prompt octopus.Prompt) *terraform.TerraformProjectVariablePrompt {
	if prompt.Label != nil || prompt.Description != nil {
		return &terraform.TerraformProjectVariablePrompt{
			Description: prompt.Description,
			Label:       prompt.Label,
			IsRequired:  prompt.Required,
		}
	}

	return nil
}
