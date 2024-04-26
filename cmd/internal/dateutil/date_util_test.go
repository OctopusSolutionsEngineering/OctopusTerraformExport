package dateutil

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"testing"
)

func TestRemoveTimezoneNil(t *testing.T) {
	result, err := RemoveTimezone(nil)

	if err != nil {
		t.Fatalf("should not have returned an error")
	}

	if result != nil {
		t.Fatalf("result should have been nil")
	}
}

func TestRemoveTimezone(t *testing.T) {
	date := "2024-04-26T09:00:00.000Z"
	result, err := RemoveTimezone(&date)

	if err != nil {
		t.Fatalf("should not have returned an error")
	}

	if strutil.EmptyIfNil(result) != "2024-04-26T09:00:00" {
		t.Fatalf("result should have been 2024-04-26T09:00:00")
	}
}
