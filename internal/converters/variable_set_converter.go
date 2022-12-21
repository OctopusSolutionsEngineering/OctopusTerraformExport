package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
	"strings"
)

type VariableSetConverter struct {
	Client client.OctopusClient
}

func (c VariableSetConverter) ToHclById(id string, parentName string) (map[string]string, error) {
	resource := model.VariableSet{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil, err
	}

	resources := map[string]string{}
	file := hclwrite.NewEmptyFile()

	for _, v := range resource.Variables {
		resourceName := parentName + "_" + util.SanitizeName(v.Name)

		terraformResource := model.TerraformProjectVariable{
			Name:           resourceName,
			Type:           "octopusdeploy_variable",
			OwnerId:        "octopusdeploy_project." + parentName + ".id",
			Value:          v.Value,
			ResourceName:   v.Name,
			Description:    v.Description,
			SensitiveValue: c.convertSecretValue(v, parentName),
			IsSensitive:    v.IsSensitive,
			Prompt: model.TerraformProjectVariablePrompt{
				Description: v.Prompt.Description,
				Label:       v.Prompt.Label,
				IsRequired:  v.Prompt.Required,
			},
		}

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
		// Unescape dollar signs because of https://github.com/hashicorp/hcl/issues/323
		resources[internal.PopulateSpaceDir+"/"+resourceName+".tf"] = strings.ReplaceAll(string(file.Bytes()), "$${", "${")
	}

	return resources, nil
}

func (c VariableSetConverter) GetResourceType() string {
	return "Variables"
}

func (c VariableSetConverter) convertSecretValue(variable model.Variable, parentName string) *string {
	if variable.IsSensitive {
		value := "${var." + parentName + ".id}"
		return &value
	}

	return nil
}
