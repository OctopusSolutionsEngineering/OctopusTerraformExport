package sliceutil

// Contains exists because go. Just because go. You know exactly what I mean...
func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func NilIfEmpty[T any](s []T) []T {
	if s == nil || len(s) == 0 {
		return nil
	}

	return s
}
