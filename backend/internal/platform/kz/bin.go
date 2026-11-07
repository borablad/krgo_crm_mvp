// Package kz — проверки идентификаторов Казахстана.
package kz

// ValidBIN проверяет БИН или ИИН: 12 цифр и контрольный разряд по алгоритму РК.
// Для контрольной суммы используются веса 1…11; если остаток равен 10 — веса сдвигаются на 2 (3…11, 1, 2);
// если и тогда остаток 10 — номер некорректен.
func ValidBIN(s string) bool {
	if len(s) != 12 {
		return false
	}
	var d [12]int
	same := true
	for i := 0; i < 12; i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return false
		}
		d[i] = int(c - '0')
		if d[i] != d[0] {
			same = false
		}
	}
	// 000000000000 проходит по контрольной сумме, но номером быть не может.
	if same {
		return false
	}
	sum := 0
	for i := 0; i < 11; i++ {
		sum += d[i] * (i + 1)
	}
	check := sum % 11
	if check == 10 {
		sum = 0
		for i := 0; i < 11; i++ {
			w := (i + 3) % 11
			if w == 0 {
				w = 11
			}
			sum += d[i] * w
		}
		check = sum % 11
		if check == 10 {
			return false
		}
	}
	return check == d[11]
}
