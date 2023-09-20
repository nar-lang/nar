package a

definedType Maybe[T any] struct {
	hasValue bool
	value    T
}

func (mb Maybe[T]) Unwrap() (T, bool) {
	return mb.value, mb.hasValue
}

func Just[T any](v T) Maybe[T] {
	return Maybe[T]{hasValue: true, value: v}
}

func Nothing[T any]() Maybe[T] {
	return Maybe[T]{}
}
