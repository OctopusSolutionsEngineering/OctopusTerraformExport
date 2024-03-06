package converters

import "testing"

func TestLimitAttributeLength(t *testing.T) {
	processor := OctopusActionProcessor{}
	properties := map[string]string{
		"MyProperty": "This is a very long property that should be limited #{Variable1} $OctopusParameters['Variable2'] get_octopusvariable \"Variable3\" get_octopusvariable('Variable4')",
	}

	processedProperties := processor.LimitPropertyLength(10, true, properties)

	if processedProperties["MyProperty"] != "This is a #{Variable1} $OctopusParameters['Variable2'] get_octopusvariable \"Variable3\" get_octopusvariable('Variable4')" {
		t.Fatalf("Property was not processed correctly")
	}
}
