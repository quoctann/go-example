package sliceutils

// Map chuyển đổi mỗi phần tử của slice theo hàm mapper
func Map[T any, R any](slice []T, mapper func(T) R) []R {
	result := make([]R, len(slice))
	for i, v := range slice {
		result[i] = mapper(v)
	}
	return result
}

// Filter giữ lại các phần tử thỏa mãn predicate
func Filter[T any](slice []T, predicate func(T) bool) []T {
	var result []T
	for _, v := range slice {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

// Find trả về phần tử đầu tiên thỏa mãn predicate (và true nếu tìm thấy)
func Find[T any](slice []T, predicate func(T) bool) (T, bool) {
	for _, v := range slice {
		if predicate(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// FindIndex trả về index của phần tử đầu tiên thỏa mãn predicate, hoặc -1 nếu không tìm thấy
func FindIndex[T any](slice []T, predicate func(T) bool) int {
	for i, v := range slice {
		if predicate(v) {
			return i
		}
	}
	return -1
}

// Some kiểm tra có ít nhất một phần tử thỏa mãn predicate không (tương đương JS some)
func Some[T any](slice []T, predicate func(T) bool) bool {
	for _, v := range slice {
		if predicate(v) {
			return true
		}
	}
	return false
}

// Every kiểm tra tất cả phần tử có thỏa mãn predicate không (tương đương JS every)
func Every[T any](slice []T, predicate func(T) bool) bool {
	for _, v := range slice {
		if !predicate(v) {
			return false
		}
	}
	return true
}

// Reduce gom slice về một giá trị duy nhất
func Reduce[T any, R any](slice []T, reducer func(acc R, curr T) R, initial R) R {
	acc := initial
	for _, v := range slice {
		acc = reducer(acc, v)
	}
	return acc
}

// ReduceWithoutInitial giống Reduce nhưng lấy phần tử đầu tiên làm initial (nếu slice rỗng thì trả về zero value)
func ReduceWithoutInitial[T any](slice []T, reducer func(acc T, curr T) T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	acc := slice[0]
	for _, v := range slice[1:] {
		acc = reducer(acc, v)
	}
	return acc, true
}

// ForEach thực thi side-effect cho từng phần tử (tương đương JS forEach)
func ForEach[T any](slice []T, callback func(T)) {
	for _, v := range slice {
		callback(v)
	}
}

// Includes kiểm tra slice có chứa value không (dùng ==, nên dùng cho kiểu comparable)
func Includes[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
