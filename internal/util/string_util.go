package util

func NilIfEmpty(input *string) *string {
	if input == nil {
		return nil
	}

	if *input == "" {
		return nil
	}

	return input
}
