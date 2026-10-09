package export

import "testing"

func TestFormatarValor(t *testing.T) {
	casos := map[float64]string{
		0:          "0,00",
		80:         "80,00",
		1234.5:     "1 234,50",
		1234567.89: "1 234 567,89",
		-45.1:      "-45,10",
	}
	for v, want := range casos {
		if got := formatarValor(v); got != want {
			t.Errorf("formatarValor(%v) = %q, want %q", v, got, want)
		}
	}
}

func TestFormatarQtd(t *testing.T) {
	casos := map[float64]string{3: "3", 1.5: "1,5", 0.125: "0,125"}
	for v, want := range casos {
		if got := formatarQtd(v); got != want {
			t.Errorf("formatarQtd(%v) = %q, want %q", v, got, want)
		}
	}
}
