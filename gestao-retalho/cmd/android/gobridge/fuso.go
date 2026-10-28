package gobridge

import (
	"fmt"
	"time"
)

// tzPosix escreve o desvio atual de `loc` em formato POSIX, o único que a
// libc do SQLite entende sem base de fusos: Maputo → "<+0200>-2" (no POSIX
// o sinal é invertido). Não leva regras de hora de verão — Moçambique não
// as usa; num país que as use vale o desvio do momento do arranque.
func tzPosix(loc *time.Location, agora time.Time) string {
	_, seg := agora.In(loc).Zone()
	min := seg / 60
	sinalNome, sinalPosix := "+", "-"
	if min < 0 {
		sinalNome, sinalPosix, min = "-", "", -min
	}
	h, m := min/60, min%60
	desvio := fmt.Sprintf("%s%d", sinalPosix, h)
	if m != 0 {
		desvio += fmt.Sprintf(":%02d", m)
	}
	if desvio == "-0" {
		desvio = "0"
	}
	return fmt.Sprintf("<%s%02d%02d>%s", sinalNome, h, m, desvio)
}
