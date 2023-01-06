package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type VariableSetConverter struct {
	Client      client.OctopusClient
	AccountsMap map[string]string
}

func (c VariableSetConverter) ToHclById(id string, parentName string) (map[string]string, error) {
	resource := octopus.VariableSet{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil, err
	}

	resources := map[string]string{}
	file := hclwrite.NewEmptyFile()

	for _, v := range resource.Variables {
		resourceName := parentName + "_" + util.SanitizeNamePointer(v.Name)

		terraformResource := terraform.TerraformProjectVariable{
			Name:           resourceName,
			Type:           "octopusdeploy_variable",
			OwnerId:        "octopusdeploy_project." + parentName + ".id",
			Value:          c.replaceAccountIds(v.Value),
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
				Description: "The secret variable value associated with the variable " + *v.Name,
			}

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)
		}

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
		resources["space_population/"+resourceName+".tf"] = string(file.Bytes())
	}

	return resources, nil
}

func (c VariableSetConverter) GetResourceType() string {
	return "Variables"
}

func (c VariableSetConverter) convertSecretValue(variable octopus.Variable, parentName string) *string {
	if variable.IsSensitive {
		value := "${var." + parentName + "}"
		return &value
	}

	return nil
}

func (c VariableSetConverter) convertPrompt(prompt octopus.Prompt) *terraform.TerraformProjectVariablePrompt {
	if prompt.Label != nil || prompt.Description != nil {
		return &terraform.TerraformProjectVariablePrompt{
			Description: prompt.Description,
			Label:       prompt.Label,
			IsRequired:  prompt.Required,
		}
	}

	return nil
}

// replaceAccountIds swaps out an account ID with the resource lookup expression
func (c VariableSetConverter) replaceAccountIds(variableValue *string) *string {
	if variableValue == nil {
		return variableValue
	}

	if val, ok := c.AccountsMap[*variableValue]; ok {
		return &val
	}

	return variableValue
}
