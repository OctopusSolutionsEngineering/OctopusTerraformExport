package hcl

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
)

func EncodeTerraformVariable(secretVariableResource terraform.TerraformVariable) *hclwrite.Block {
	block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
	WriteUnquotedAttribute(block, "type", "string")
	return block
}
