package sanitizer

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
)

// SanitizeMap takes a map returned by the Octopus API, and replaces any sensitive value references with a placeholder
func SanitizeMap(parentName string, actionName string, input map[string]any) (map[string]string, []terraform.TerraformVariable) {
	variables := []terraform.TerraformVariable{}
	fixedMap := map[string]string{}
	for k, v := range input {
		if _, ok := v.(string); ok {
			fixedMap[k] = fmt.Sprintf("%v", v)
		} else {
			variableName := SanitizeName(parentName + "_" + actionName + "_" + k)

			fixedMap[k] = "${var." + variableName + "}"

			secretVariableResource := terraform.TerraformVariable{
				Name:        variableName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "Sensitive value for property " + k,
				Default:     strutil.StrPointer("replace me with a password"),
			}

			variables = append(variables, secretVariableResource)

		}
	}
	return fixedMap, variables
}
