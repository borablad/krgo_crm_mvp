package kz

import (
	"fmt"
	"math/rand"
	"testing"
)

// checkDigit — независимая реализация алгоритма РК для проверки ValidBIN: -1, если номер не может существовать.
func checkDigit(first11 string) int {
	w1 := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	w2 := []int{3, 4, 5, 6, 7, 8, 9, 10, 11, 1, 2}
	sum := func(w []int) int {
		s := 0
		for i := 0; i < 11; i++ {
			s += int(first11[i]-'0') * w[i]
		}
		return s % 11
	}
	c := sum(w1)
	if c == 10 {
		c = sum(w2)
		if c == 10 {
			return -1
		}
	}
	return c
}

func TestValidBIN_Known(t *testing.T) {
	if !ValidBIN("940140000385") { // БИН АО «Казпочта»
		t.Error("940140000385 должен быть корректным")
	}
	for _, v := range []string{"", "12345678901", "1234567890123", "94014000038x", "940140000384", "000000000000", "111111111111"} {
		if ValidBIN(v) {
			t.Errorf("%q не должен проходить проверку", v)
		}
	}
}

func TestValidBIN_Generated(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	secondBranch := 0
	for n := 0; n < 2000; n++ {
		prefix := fmt.Sprintf("%011d", rnd.Int63n(100_000_000_000))
		c := checkDigit(prefix)
		if c < 0 {
			for d := 0; d < 10; d++ {
				if ValidBIN(prefix + fmt.Sprint(d)) {
					t.Fatalf("%s%d: номер не может существовать, но принят", prefix, d)
				}
			}
			continue
		}
		s := 0
		for i := 0; i < 11; i++ {
			s += int(prefix[i]-'0') * (i + 1)
		}
		if s%11 == 10 {
			secondBranch++
		}
		for d := 0; d < 10; d++ {
			got := ValidBIN(prefix + fmt.Sprint(d))
			if got != (d == c) {
				t.Fatalf("%s%d: ValidBIN=%v, ожидалось %v (контрольная %d)", prefix, d, got, d == c, c)
			}
		}
	}
	if secondBranch == 0 {
		t.Fatal("ни один номер не задействовал вторую таблицу весов")
	}
}
