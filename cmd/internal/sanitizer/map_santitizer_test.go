package sanitizer

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"testing"
)

func TestMapSanitizer(t *testing.T) {
	sanitizedMap, _ := SanitizeMap("parent",
		"actionname",
		map[string]any{
			"input":  "test",
			"input2": octopus.Variable{},
		})

	if sanitizedMap["input"] != "test" {
		t.Fatal("String should be passed through with o changes")
	}

	if sanitizedMap["input2"] != "${var.parent_actionname_input2}" {
		t.Fatal("Object should be replaced with placeholder")
	}
}
