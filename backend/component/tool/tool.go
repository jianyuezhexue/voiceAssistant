package tool

// 取指针
func Of[T any](t T) *T {
	return &t
}

// 三元运算符
func Ternary[T any](condition bool, trueVal, falseVal T) T {
	if condition {
		return trueVal
	}
	return falseVal
}
