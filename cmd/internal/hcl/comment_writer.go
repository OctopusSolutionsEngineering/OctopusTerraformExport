package hcl

import (
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
)

func WriteImportComments(baseUrl string, resourceType string, tfResourceType string, resourceName string, tfResourceName string) []*hclwrite.Token {
	return []*hclwrite.Token{{
		Type: hclsyntax.TokenComment,
		Bytes: []byte("# Import existing resources with the following commands:\n" +
			"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + resourceType + " | jq -r '.Items[] | select(.Name==\"" + resourceName + "\") | .Id')\n" +
			"# terraform import " + tfResourceType + "." + tfResourceName + " ${RESOURCE_ID}\n"),
		SpacesBefore: 0,
	}}
}
