package kz

import "testing"

func TestValidBIN(t *testing.T) {
	valid := []string{
		"940140000385", // БИН Казпочты
		"000140000019", // БИН НацБанка РК
	}
	for _, v := range valid {
		if !ValidBIN(v) {
			t.Errorf("%s должен быть корректным", v)
		}
	}
	invalid := []string{"", "12345678901", "1234567890123", "94014000038x", "940140000384", "000000000000"}
	for _, v := range invalid {
		if ValidBIN(v) {
			t.Errorf("%q не должен проходить проверку", v)
		}
	}
}
