package impressao

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"testing"
)

func TestTalaoTextoSimples(t *testing.T) {
	tl := NovoTalao(32).Titulo("MESA 5").Separador().Par("2 x Heineken", "300,00", false).
		Par("1 x Um nome de produto muito comprido mesmo", "1.250,00", true)
	linhas := strings.Split(strings.TrimRight(tl.TextoSimples(), "\n"), "\n")
	for _, l := range linhas {
		if n := len([]rune(l)); n > 32 {
			t.Errorf("linha com %d colunas: %q", n, l)
		}
	}
	if !strings.HasSuffix(linhas[2], "300,00") || !strings.HasPrefix(linhas[2], "2 x Heineken") {
		t.Errorf("par mal alinhado: %q", linhas[2])
	}
	if !strings.HasSuffix(linhas[len(linhas)-1], "1.250,00") {
		t.Errorf("valor do par comprido não ficou na última linha: %q", linhas)
	}
}

func TestESCPOSAcentosECorte(t *testing.T) {
	b := NovoTalao(48).Texto("Feijoada à moda, porção").ESCPOS()
	if !bytes.HasPrefix(b, append(append([]byte{}, cmdIniciar...), cmdPagina860...)) {
		t.Fatal("falta o arranque e a página de código")
	}
	if !bytes.HasSuffix(b, cmdAvancarCorte) {
		t.Fatal("falta o corte")
	}
	// Em PC860: à = 0x85, ç = 0x87, ã = 0x84.
	for _, c := range []byte{0x85, 0x87, 0x84} {
		if !bytes.Contains(b, []byte{c}) {
			t.Errorf("falta o byte %#x", c)
		}
	}
}

func TestImpressoraRedeEnvia(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	recebido := make(chan []byte, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		b, _ := io.ReadAll(c)
		recebido <- b
	}()
	porta := ln.Addr().(*net.TCPAddr).Port
	if err := (ImpressoraRede{}).Enviar(context.Background(), "127.0.0.1", porta, []byte("ola")); err != nil {
		t.Fatal(err)
	}
	if got := <-recebido; string(got) != "ola" {
		t.Fatalf("recebido %q", got)
	}
}
