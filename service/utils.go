package service

func SplitSlice[T any](s []T, size int) [][]T {
	var sl [][]T
	for len(s) > size {
		sl = append(sl, s[:size])
		s = s[size:]
	}
	if len(s) > 0 {
		sl = append(sl, s)
	}
	return sl
}
