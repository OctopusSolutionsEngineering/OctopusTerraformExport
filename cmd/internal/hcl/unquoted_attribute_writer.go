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
func WriteActionProperties(block *hclwrite.Block, step int, action int, properties map[string]string) {
	block.Body().Blocks()[step].Body().Blocks()[action].Body().SetAttributeTraversal("properties", hcl.Traversal{
		hcl.TraverseRoot{Name: extractJsonAsMap(properties)},
	})
}

func extractJsonAsMap(properties map[string]string) string {
	output := "{"

	for key, value := range properties {
		output += "\n        \"" + key + "\" = " + jsonStringToHcl(fmt.Sprint(value))
	}

	output += "\n      }"

	return output
}

func jsonStringToHcl(value string) string {
	jsonMap := map[string]any{}
	jsonMapError := json.Unmarshal([]byte(value), &jsonMap)

	jsonArray := []any{}
	jsonArrayError := json.Unmarshal([]byte(value), &jsonArray)

	if jsonMapError == nil {
		return "jsonencode(" + mapToHclMap(jsonMap) + ")"
	} else if jsonArrayError == nil {
		return "jsonencode(" + arrayToHclMap(jsonArray) + ")"
	} else {
		return "\"" + encodeString(value) + "\""
	}
}

func anyToHcl(value any) string {
	if mapItem, ok := value.(map[string]any); ok {
		return mapToHclMap(mapItem)
	} else if arrayItem, ok := value.([]any); ok {
		return arrayToHclMap(arrayItem)
	} else {
		return "\"" + encodeString(fmt.Sprint(value)) + "\""
	}
}

func mapToHclMap(jsonMap map[string]any) string {
	output := "{"
	for k, v := range jsonMap {
		output += "\n        \"" + k + "\" = " + anyToHcl(v)
	}
	output += "\n        }"
	return output
}

func arrayToHclMap(jsonArray []any) string {
	output := "["
	for _, v := range jsonArray {
		output += "\n        " + anyToHcl(v) + ","
	}
	output += "\n        ]"
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
