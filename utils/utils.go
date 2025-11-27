package utils

func ToNonPointer[T any](pointer *T) T {
	if pointer != nil {
		return *pointer
	}
	return *new(T)
}
