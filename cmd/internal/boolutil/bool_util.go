package boolutil

// FalseIfNil converts a bool pointer to a bool, retuning false if the input is nil
func FalseIfNil(input *bool) bool {
	if input == nil {
		return false
	}

	return *input
}
