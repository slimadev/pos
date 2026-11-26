package impressao

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"
)

// Impressor envia bytes já codificados para uma impressora. Os testes usam
// um falso; em produção é a ImpressoraRede.
type Impressor interface {
	Enviar(ctx context.Context, endereco string, porta int, dados []byte) error
}

// ImpressoraRede fala com impressoras térmicas ligadas por cabo de rede
// (RJ45), na porta "raw" 9100 — o modo que a TM-T20 e as Xprinter trazem.
type ImpressoraRede struct {
	Timeout time.Duration
}

func (p ImpressoraRede) Enviar(ctx context.Context, endereco string, porta int, dados []byte) error {
	timeout := p.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	if porta == 0 {
		porta = 9100
	}
	alvo := net.JoinHostPort(endereco, strconv.Itoa(porta))
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", alvo)
	if err != nil {
		return fmt.Errorf("impressora %s não responde (ligada e na rede?)", alvo)
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	if _, err := conn.Write(dados); err != nil {
		return fmt.Errorf("enviar para a impressora %s: %w", alvo, err)
	}
	return nil
}
