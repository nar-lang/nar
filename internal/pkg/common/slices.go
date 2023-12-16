package common

import (
	"fmt"
	"strings"
)

func Range(min, max int) []int {
	result := make([]int, max-min)
	for i := range result {
		result[i] = i + min
	}
	return result
}

func Map[I, O any](p func(I) O, xs []I) []O {
	result := make([]O, len(xs))
	for i, x := range xs {
		result[i] = p(x)
	}
	return result
}

func MapIf[I, O any](p func(I) (O, bool), xs []I) []O {
	result := make([]O, 0, len(xs))
	for _, x := range xs {
		if r, ok := p(x); ok {
			result = append(result, r)
		}
	}
	return result
}

func ConcatMap[I, O any](p func(I) []O, xs []I) []O {
	result := make([]O, 0, len(xs))
	for _, x := range xs {
		result = append(result, p(x)...)
	}
	return result
}

func Repeat[T any](x T, n int) []T {
	result := make([]T, n)
	for i := range result {
		result[i] = x
	}
	return result
}

func Any[T any](p func(T) bool, xs []T) bool {
	for _, x := range xs {
		if p(x) {
			return true
		}
	}
	return false
}

func Find[T any](p func(T) bool, xs []T) (T, bool) {
	for _, x := range xs {
		if p(x) {
			return x, true
		}
	}

	var x T
	return x, false
}

func Join[T fmt.Stringer](xs []T, sep string) string {
	return strings.Join(Map(func(x T) string { return x.String() }, xs), sep)
}
