package intutil

// ZeroIfNil converts a int pointer to an int, retuning 0 if the input is nil
func ZeroIfNil(input *int) int {
	if input == nil {
		return 0
	}

	return *input
}
