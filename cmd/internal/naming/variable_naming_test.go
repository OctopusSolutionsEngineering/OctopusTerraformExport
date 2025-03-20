package naming

import (
	"testing"

	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
)

func TestVariableSecretName(t *testing.T) {
	variable := octopus.Variable{Id: "test-id"}
	expected := "variable_6cc41d5ec590ab78cccecf81ef167d418c309a4598e8e45fef78039f7d9aa9fe_sensitive_value" // Replace with the actual expected hash
	result := VariableSecretName(variable)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestVariableValueName(t *testing.T) {
	variable := octopus.Variable{Id: "test-id"}
	expected := "variable_6cc41d5ec590ab78cccecf81ef167d418c309a4598e8e45fef78039f7d9aa9fe_value" // Replace with the actual expected hash
	result := VariableValueName(variable)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}
