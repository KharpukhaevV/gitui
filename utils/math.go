package utils

// Min возвращает минимальное из двух чисел
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max возвращает максимальное из двух чисел
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
