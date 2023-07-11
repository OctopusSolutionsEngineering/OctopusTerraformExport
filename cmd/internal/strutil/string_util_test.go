package strutil

import "testing"

func TestNilIfEmptyPointer(t *testing.T) {
	emptyString := ""
	notEmptyString := "test"

	if NilIfEmptyPointer(&emptyString) != nil {
		t.Fatalf("result should have been nil")
	}

	if NilIfEmptyPointer(nil) != nil {
		t.Fatalf("result should have been nil")
	}

	if NilIfEmptyPointer(&notEmptyString) == nil {
		t.Fatalf("result should not have been nil")
	}
}

func TestEmptyIfNil(t *testing.T) {
	emptyString := ""
	notEmptyString := "test"

	if EmptyIfNil(&emptyString) != "" {
		t.Fatalf("result should have been nil")
	}

	if EmptyIfNil(nil) != "" {
		t.Fatalf("result should have been nil")
	}

	if EmptyIfNil(&notEmptyString) == "" {
		t.Fatalf("result should not have been nil")
	}
}

func TestFalseIfNil(t *testing.T) {
	trueBool := true
	falseBool := false

	if FalseIfNil(nil) {
		t.Fatalf("result should have been false")
	}

	if !FalseIfNil(&trueBool) {
		t.Fatalf("result should have been true")
	}

	if FalseIfNil(&falseBool) {
		t.Fatalf("result should not have been false")
	}
}

func TestNilIfEmpty(t *testing.T) {
	emptyString := ""
	notEmptyString := "test"

	if NilIfEmpty(emptyString) != nil {
		t.Fatalf("result should have been nil")
	}

	if NilIfEmpty(notEmptyString) == nil {
		t.Fatalf("result should not have been nil")
	}
}

func TestDefaultIfEmpty(t *testing.T) {
	if DefaultIfEmpty("", "default") != "default" {
		t.Fatalf("result should have been default")
	}

	if DefaultIfEmpty("notempty", "default") != "notempty" {
		t.Fatalf("result should not have been notempty")
	}
}

func TestDefaultIfEmptyOrNil(t *testing.T) {
	emptyString := ""
	notEmptyString := "test"

	if DefaultIfEmptyOrNil(&emptyString, "default") != "default" {
		t.Fatalf("result should have been default")
	}

	if DefaultIfEmptyOrNil(nil, "default") != "default" {
		t.Fatalf("result should have been default")
	}

	if DefaultIfEmptyOrNil(&notEmptyString, "default") != notEmptyString {
		t.Fatalf("result should not have been notempty")
	}
}

func TestNilIfFalse(t *testing.T) {
	if NilIfFalse(false) != nil {
		t.Fatalf("result should have been nil")
	}

	if !*NilIfFalse(true) {
		t.Fatalf("result should not have been true")
	}
}

func TestUnEscapeDollar(t *testing.T) {
	unescapedMap := UnEscapeDollar(map[string]string{
		"entry1": "\"$${var.value}\"",
		"entry2": "$${var.value}",
		"entry3": "\"value\"",
		"entry4": "value",
		"entry5": "\"$${var.value}blah$${var.value}\"",
		"entry6": "default     = \"$${var.project_noopterraform_description_prefix}NoOpTerraform$${var.project_noopterraform_description_suffix}\"",
		"entry7": "environments = [\"$${data.octopusdeploy_environments.test.environments[0].id}\"]",
	})

	if unescapedMap["entry1"] != "\"${var.value}\"" {
		t.Fatalf("result should have been \"${var.value}\"")
	}

	if unescapedMap["entry2"] != "${var.value}" {
		t.Fatalf("result should have been ${var.value}")
	}

	if unescapedMap["entry3"] != "\"value\"" {
		t.Fatalf("result should have been \"value\"")
	}

	if unescapedMap["entry4"] != "value" {
		t.Fatalf("result should have been value")
	}

	if unescapedMap["entry5"] != "\"${var.value}blah${var.value}\"" {
		t.Fatalf("result should have been \"${var.value}\"blah\"${var.value}\"")
	}

	if unescapedMap["entry6"] != "default     = \"${var.project_noopterraform_description_prefix}NoOpTerraform${var.project_noopterraform_description_suffix}\"" {
		t.Fatalf("result should have been default     = \"${var.project_noopterraform_description_prefix}NoOpTerraform${var.project_noopterraform_description_suffix}\"")
	}

	if unescapedMap["entry7"] != "environments = [\"${data.octopusdeploy_environments.test.environments[0].id}\"]" {
		t.Fatalf("result should have been environments = [\"${data.octopusdeploy_environments.test.environments[0].id}\"]")
	}
}

func TestParseBool(t *testing.T) {
	if ParseBool("false") {
		t.Fatalf("result should have been false")
	}

	if ParseBool("FALSE") {
		t.Fatalf("result should have been false")
	}

	if ParseBool("blah") {
		t.Fatalf("result should have been false")
	}

	if !ParseBool("true") {
		t.Fatalf("result should have been true")
	}

	if !ParseBool("TRUE") {
		t.Fatalf("result should have been true")
	}
}

func TestParseBoolPointer(t *testing.T) {
	falseString := "false"
	falseUpperString := "FALSE"
	trueString := "true"
	trueUpperString := "TRUE"
	nonsenseString := "blah"

	if *ParseBoolPointer(&falseString) {
		t.Fatalf("result should have been false")
	}

	if *ParseBoolPointer(&falseUpperString) {
		t.Fatalf("result should have been false")
	}

	if *ParseBoolPointer(&nonsenseString) {
		t.Fatalf("result should have been false")
	}

	if !*ParseBoolPointer(&trueString) {
		t.Fatalf("result should have been true")
	}

	if !*ParseBoolPointer(&trueUpperString) {
		t.Fatalf("result should have been true")
	}
}

func TestEnsureSuffix(t *testing.T) {
	if EnsureSuffix("test!", "!") != "test!" {
		t.Fatalf("result should have been test!")
	}

	if EnsureSuffix("test", "!") != "test!" {
		t.Fatalf("result should have been test!")
	}

	if EnsureSuffix("test", "blah") != "testblah" {
		t.Fatalf("result should have been testblah")
	}

	if EnsureSuffix("test! ", "!") != "test! !" {
		t.Fatalf("result should have been test! !")
	}

	if EnsureSuffix(" ", "!") != " !" {
		t.Fatalf("result should have been !")
	}
}
