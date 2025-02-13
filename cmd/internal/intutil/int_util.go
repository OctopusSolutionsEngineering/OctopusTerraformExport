package intutil

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"strconv"
)

// ZeroIfNil converts a int pointer to an int, retuning 0 if the input is nil
func ZeroIfNil(input *int) int {
	if input == nil {
		return 0
	}

	return *input
}

func ParseIntPointer(input *string) (*int, error) {
	if input == nil {
		return nil, nil
	}

	num, err := strconv.Atoi(strutil.EmptyIfNil(input))

	if err != nil {
		return nil, err
	}

	return &num, err
}
