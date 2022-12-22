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
	Client client.OctopusClient
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
		resourceName := parentName + "_" + util.SanitizeName(v.Name)

		terraformResource := terraform.TerraformProjectVariable{
			Name:           resourceName,
			Type:           "octopusdeploy_variable",
			OwnerId:        "octopusdeploy_project." + parentName + ".id",
			Value:          v.Value,
			ResourceName:   v.Name,
			Description:    v.Description,
			SensitiveValue: c.convertSecretValue(v, parentName),
			IsSensitive:    v.IsSensitive,
			Prompt:         c.convertPrompt(v.Prompt),
		}

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
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
