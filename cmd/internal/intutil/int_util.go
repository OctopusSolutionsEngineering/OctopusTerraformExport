package intutil

func ZeroIfNil(input *int) int {
	if input == nil {
		return 0
	}

	return *input
}
