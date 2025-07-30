package variables

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/hashicorp/hcl2/hclwrite"
)

// TerraformVariableWriter provides functions to create Terraform variables for sensitive values in Octopus.
type TerraformVariableWriter interface {
	WriteTerraformVariablesForSecret(resourceType string, file *hclwrite.File, variable octopus.NamedResource, dependencies *data.ResourceDetailsCollection) *string
}
