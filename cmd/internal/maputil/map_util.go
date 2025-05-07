package maputil

func NilIfEmptyMap(input map[string]string) *map[string]string {
	if len(input) == 0 {
		return nil
	}

	return &input
}
