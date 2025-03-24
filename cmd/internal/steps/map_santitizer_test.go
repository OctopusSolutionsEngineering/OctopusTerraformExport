package steps

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"testing"
)

func TestMapSanitizer(t *testing.T) {
	sanitizedMap, _ := MapSanitizer{}.SanitizeMap(octopus.NameId{
		Id:      "parent",
		SpaceId: "",
		Name:    "parentName",
	},
		octopus.Action{
			Id: "actionname",
		},
		map[string]any{
			"input":  "test",
			"input2": octopus.Variable{},
		}, nil)

	if sanitizedMap["input"] != "test" {
		t.Fatal("String should be passed through with o changes")
	}

	if sanitizedMap["input2"] != "${var.action_4481fe3a58f14368f78761e4acd1bbcaabcb3e35e596829aec0a37c086d52df8_sensitive_value}" {
		t.Fatal("Object should be replaced with placeholder")
	}
}
