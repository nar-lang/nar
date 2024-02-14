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

func FlatMap[I, O any](p func(I) []O, xs []I) []O {
	result := make([]O, 0, len(xs))
	for _, x := range xs {
		result = append(result, p(x)...)
	}
	return result
}

func ForError[I any](p func(I) error, xs []I) error {
	for _, x := range xs {
		if err := p(x); err != nil {
			return err
		}
	}
	return nil
}

func MapError[I, O any](p func(I) (O, error), xs []I) ([]O, error) {
	result := make([]O, len(xs))
	for i, x := range xs {
		r, err := p(x)
		if err != nil {
			return nil, err
		}
		result[i] = r
	}
	return result, nil
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

func MapIfError[I, O any](p func(I) (O, bool, error), xs []I) ([]O, error) {
	result := make([]O, 0, len(xs))
	for _, x := range xs {
		r, ok, err := p(x)
		if err != nil {
			return nil, err
		}
		if ok {
			result = append(result, r)
		}
	}
	return result, nil
}

func ConcatMap[I, O any](p func(I) []O, xs []I) []O {
	result := make([]O, 0, len(xs))
	for _, x := range xs {
		result = append(result, p(x)...)
	}
	return result
}

func ConcatMapError[I, O any](p func(I) ([]O, error), xs []I) ([]O, error) {
	result := make([]O, 0, len(xs))
	for _, x := range xs {
		r, err := p(x)
		if err != nil {
			return nil, err
		}
		result = append(result, r...)
	}
	return result, nil
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

func AnyError[T any](p func(T) (bool, error), xs []T) (bool, error) {
	hasTrue := false
	for _, x := range xs {
		r, err := p(x)
		if err != nil {
			return false, err
		}
		hasTrue = hasTrue || r
	}
	return hasTrue, nil
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

func Fold[T, A any](p func(T, A) A, acc A, xs []T) A {
	result := acc
	for _, x := range xs {
		result = p(x, result)
	}
	return result
}

func Keys[K comparable, V any](m map[K]V) []K {
	result := make([]K, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

func Values[K comparable, V any](m map[K]V) []V {
	result := make([]V, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}

func Filter[T any](p func(T) bool, xs []T) []T {
	result := make([]T, 0, len(xs))
	for _, x := range xs {
		if p(x) {
			result = append(result, x)
		}
	}
	return result
}

func MergeErrors(errors ...error) error {
	nonNil := Filter(func(e error) bool { return e != nil }, errors)
	if len(nonNil) == 0 {
		return nil
	}
	s := Map(func(e error) string { return e.Error() }, nonNil)
	return fmt.Errorf(strings.Join(s, "\n"))
}
