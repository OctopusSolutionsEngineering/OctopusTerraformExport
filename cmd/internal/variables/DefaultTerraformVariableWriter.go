package variables

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/naming"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
)

type DefaultTerraformVariableWriter struct {
	ExcludeTerraformVariables   bool
	DummySecretVariableValues   bool
	DefaultSecretVariableValues bool
	DummySecretGenerator        dummy.DummySecretGenerator
}

func (c DefaultTerraformVariableWriter) WriteTerraformVariablesForSecret(resourceType string, file *hclwrite.File, variable octopus.NamedResource, dependencies *data.ResourceDetailsCollection) *string {
	// We don't know the value of secrets, so the value is just nil
	if c.ExcludeTerraformVariables {
		return nil
	}

	var defaultValue *string = nil

	variableName := naming.VariableSecretName(variable)

	// Dummy values are used if we are not also replacing the variable with an octostache template
	// with the DefaultSecretVariableValues option.
	if c.DummySecretVariableValues && !c.DefaultSecretVariableValues {
		defaultValue = c.DummySecretGenerator.GetDummySecret()
		dependencies.AddDummy(data.DummyVariableReference{
			VariableName: variableName,
			ResourceName: variable.GetName(),
			ResourceType: resourceType,
		})
	}

	secretVariableResource := terraform.TerraformVariable{
		Name:        variableName,
		Type:        "string",
		Nullable:    true,
		Sensitive:   true,
		Description: "The secret variable value associated with the variable " + variable.GetName(),
		Default:     defaultValue,
	}

	block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")

	// If we are writing an octostache template, we need to have any string escaped for inclusion in a terraform
	// string. JSON escaping will get us most of the way there. We also need to escape any terraform syntax, which
	// unfortunately is easier said than done as there appears to be no way to write a double dollar sign with
	// the HCL serialization library, so we need to get a little creative.
	if c.DefaultSecretVariableValues {
		hcl.WriteUnquotedAttribute(block, "default", "<<EOT\n#{"+variable.GetName()+" | Replace \"([$])([{])\" \"$1$1$2\" | Replace \"([%])([{])\" \"$1$1$2\"}\nEOT")
	}

	file.Body().AppendBlock(block)

	return c.convertSecretValue(variableName)
}

func (c *DefaultTerraformVariableWriter) convertSecretValue(resourceName string) *string {
	// The heredoc string introduces a line break at the end of the string. We remove it here.
	// See https://discuss.hashicorp.com/t/trailing-new-line-in-key-vault-after-using-heredoc-syntax/14561
	if c.DefaultSecretVariableValues {
		value := "replace(var." + resourceName + ", \"/\\n$/\", \"\")"
		return &value
	}

	value := "var." + resourceName
	return &value
}
