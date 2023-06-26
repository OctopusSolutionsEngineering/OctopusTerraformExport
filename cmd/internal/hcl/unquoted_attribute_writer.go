package hcl

import (
	"encoding/json"
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"regexp"
	"strings"
)

// WriteUnquotedAttribute uses the example from https://github.com/hashicorp/hcl/issues/442
// to add an unquoted attribute to a block
func WriteUnquotedAttribute(block *hclwrite.Block, attrName string, attrValue string) {
	block.Body().SetAttributeTraversal(attrName, hcl.Traversal{
		hcl.TraverseRoot{Name: attrValue},
	})
}

// WriteLifecycleAllAttribute writes a lifecycle block with ignore_changes set to all
func WriteLifecycleAllAttribute(block *hclwrite.Block) {
	ignoreAll := terraform.TerraformLifecycleAllMetaArgument{}
	lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
	WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "all")
	block.Body().AppendBlock(lifecycleBlock)
}

// WriteLifecyclePostCondition writes a lifecycle block with a postcondition
func WriteLifecyclePostCondition(block *hclwrite.Block, errorMessage string, condition string) {
	postCondition := terraform.TerraformLifecyclePostCondition{
		ErrorMessage: errorMessage,
	}
	postConditionBlock := gohcl.EncodeAsBlock(postCondition, "postcondition")
	WriteUnquotedAttribute(postConditionBlock, "condition", condition)

	lifecycle := terraform.TerraformLifecycleMetaArgument{}
	lifecycleBlock := gohcl.EncodeAsBlock(lifecycle, "lifecycle")
	lifecycleBlock.Body().AppendBlock(postConditionBlock)
	block.Body().AppendBlock(lifecycleBlock)
}

// WriteActionProperties is used to pretty print the properties of an action, writing a multiline map for the properties,
// and extracting JSON blobs as maps for easy reading.
func WriteActionProperties(block *hclwrite.Block, stepName string, actionName string, properties map[string]string) {
	for _, stepBlock := range block.Body().Blocks() {
		stepNameTokens := hclwrite.Tokens{}
		blockStepNameAttribute := stepBlock.Body().GetAttribute("name")

		if blockStepNameAttribute == nil {
			continue
		}

		blockStepName := getAttributeValue(blockStepNameAttribute.BuildTokens(stepNameTokens))
		if blockStepName == stepName {
			for _, actionBlock := range stepBlock.Body().Blocks() {
				actionNameTokens := hclwrite.Tokens{}
				blockActionNameAttibute := actionBlock.Body().GetAttribute("name")

				if blockActionNameAttibute == nil {
					continue
				}

				blockActionName := getAttributeValue(blockActionNameAttibute.BuildTokens(actionNameTokens))
				if blockActionName == actionName {
					actionBlock.Body().SetAttributeTraversal("properties", hcl.Traversal{
						hcl.TraverseRoot{Name: extractJsonAsMap(properties)},
					})
					break
				}
			}
			break
		}
	}
}

func getAttributeValue(tokens hclwrite.Tokens) string {
	for _, token := range tokens {
		if token.Type == hclsyntax.TokenQuotedLit {
			return string(token.Bytes)
		}
	}

	return ""
}

func extractJsonAsMap(properties map[string]string) string {
	output := "{"

	for key, value := range properties {
		output += "\n        \"" + key + "\" = " + jsonStringToHcl(value)
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
	if len(jsonMap) != 0 {
		output += "\n        "
	}
	output += "        }"
	return output
}

func arrayToHclMap(jsonArray []any) string {
	output := "["
	for _, v := range jsonArray {
		output += "\n        " + anyToHcl(v) + ","
	}
	if len(jsonArray) != 0 {
		output += "\n        "
	}
	output += "]"
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

func IsInterpolation(value string) bool {
	return strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}")
}

func RemoveInterpolation(value string) string {
	value = strings.Replace(value, "${", "", -1)
	value = strings.Replace(value, "}", "", -1)
	return value
}

func RemoveId(value string) string {
	regex := regexp.MustCompile(`\.id$`)
	value = regex.ReplaceAllString(value, "")
	return value
}
