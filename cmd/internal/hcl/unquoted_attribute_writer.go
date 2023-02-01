package hcl

import (
	"encoding/json"
	"fmt"
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

// WriteActionProperties is used to pretty print the properties of an action, writing a multiline map for the properties,
// and extracting JSON blobs as maps for easy reading.
func WriteActionProperties(block *hclwrite.Block, step int, action int, properties map[string]any) {
	block.Body().Blocks()[step].Body().Blocks()[action].Body().SetAttributeTraversal("properties", hcl.Traversal{
		hcl.TraverseRoot{Name: extractJsonAsMap(properties)},
	})
}

func extractJsonAsMap(properties map[string]any) string {
	output := "{"

	for key, value := range properties {
		resource := map[string]any{}
		err := json.Unmarshal([]byte(fmt.Sprint(value)), &resource)
		if err == nil {
			output += "jsonencode({\n"
			for nestedKey, nestedValue := range resource {
				output += "\n        \"" + nestedKey + "\" = " + "\"" + encodeString(fmt.Sprint(nestedValue)) + "\""
			}
			output += "\n})"
		} else {
			output += "\n        \"" + key + "\" = " + "\"" + encodeString(fmt.Sprint(value)) + "\""
		}
	}

	output += "\n      }"

	return output
}

// encodeString assumes that HCL strings are escaped like JSON strings
func encodeString(value string) string {
	b, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	// Trim the beginning and trailing " character
	return string(b[1 : len(b)-1])
}
