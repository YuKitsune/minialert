package util

func HasMatching[T any](ts []T, fn func(v T) bool) bool {
	for _, t := range ts {
		if fn(t) {
			return true
		}
	}

	return false
}

func RemoveMatches[T any](s []T, match func(t T) bool) []T {
	for i, t := range s {
		if match(t) {
			s[i] = s[len(s)-1]
			s = s[:len(s)-1]
		}
	}

	return s
}

func FindMatching[T any](s []T, match func(t T) bool) (*T, bool) {
	for i, t := range s {
		if match(t) {
			return &s[i], true
		}
	}

	return nil, false
}
