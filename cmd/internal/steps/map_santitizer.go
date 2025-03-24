package steps

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/naming"
)

type MapSanitizer struct {
	DummySecretGenerator      dummy.DummySecretGenerator
	DummySecretVariableValues bool
}

// SanitizeMap takes a map returned by the Octopus API, and replaces any sensitive value references with a placeholder
func (c MapSanitizer) SanitizeMap(parent octopus.NamedResource, action octopus.Action, input map[string]any, dependencies *data.ResourceDetailsCollection) (map[string]string, []terraform.TerraformVariable) {
	variables := []terraform.TerraformVariable{}
	fixedMap := map[string]string{}
	for k, v := range input {
		if _, ok := v.(string); ok {
			fixedMap[k] = fmt.Sprintf("%v", v)
		} else {
			variableName := naming.DeploymentProcessPropertySecretName(parent, action, k)

			fixedMap[k] = "${var." + variableName + "}"

			var defaultValue *string = nil

			if c.DummySecretVariableValues {
				defaultValue = c.DummySecretGenerator.GetDummySecret()
				dependencies.AddDummy(data.DummyVariableReference{
					VariableName: variableName,
					ResourceName: parent.GetName(),
					ResourceType: "DeploymentProcesses",
				})
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        variableName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "Sensitive value for property " + k,
				Default:     defaultValue,
			}

			variables = append(variables, secretVariableResource)

		}
	}
	return fixedMap, variables
}
