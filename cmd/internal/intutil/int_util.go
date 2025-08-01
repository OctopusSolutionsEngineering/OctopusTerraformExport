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

// NilIfZero converts a int  to an int pointer, retuning nil if the input is 0
func NilIfZero(input int) *int {
	if input == 0 {
		return nil
	}

	return &input
}

func NilIfTrue(input int, nilValue bool) *int {
	if nilValue {
		return nil
	}

	return &input
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
