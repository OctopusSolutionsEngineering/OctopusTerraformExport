package dateutil

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"time"
)

// convertDate serializes dates and times without timezone information, as the TF
// provider will only parse "plain" date time strings when processing resources
// like triggers
func RemoveTimezone(octopusDate *string) (*string, error) {
	if octopusDate == nil {
		return nil, nil
	}

	parsedDate, err := time.Parse("2006-01-02T15:04:05.000Z07:00", strutil.EmptyIfNil(octopusDate))

	if err != nil {
		return nil, err
	}

	return strutil.StrPointer(parsedDate.Format("2006-01-02T15:04:05")), nil
}
