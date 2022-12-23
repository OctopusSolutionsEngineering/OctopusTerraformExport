package util

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclwrite"
)

// WriteUnquotedAttribute uses the example from https://github.com/hashicorp/hcl/issues/442
// to add an unquoted attribute to a block
func WriteUnquotedAttribute(block *hclwrite.Block, attrName string, attrValue string) {
	block.Body().SetAttributeTraversal(attrName, hcl.Traversal{
		hcl.TraverseRoot{Name: attrValue},
	})
}
