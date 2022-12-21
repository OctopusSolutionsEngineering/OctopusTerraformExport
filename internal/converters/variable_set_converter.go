package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
	"github.com/zclconf/go-cty/cty"
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
			SensitiveValue: c.convertSecretValue(v, resourceName),
			IsSensitive:    v.IsSensitive,
			Prompt: model.TerraformProjectVariablePrompt{
				Description: v.Prompt.Description,
				Label:       v.Prompt.Label,
				IsRequired:  v.Prompt.Required,
			},
		}

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
		resources[internal.PopulateSpaceDir+"/"+resourceName+".tf"] = string(file.Bytes())
	}

	return resources, nil
}

func (c VariableSetConverter) GetResourceType() string {
	return "Variables"
}

func (c VariableSetConverter) convertSecretValue(variable model.Variable, resourceName string) hclsyntax.Expression {
	if variable.IsSensitive {
		return &hclsyntax.LiteralValueExpr{
			Val: cty.StringVal("var." + resourceName),
		}
	}

	return nil
}
