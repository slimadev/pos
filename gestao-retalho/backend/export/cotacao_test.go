package export

import (
	"bytes"
	"os"
	"regexp"
	"testing"
	"time"
)

func TestGerarCotacaoPDF_UmaPaginaA4ComLogo(t *testing.T) {
	logo := os.Getenv("COTACAO_LOGO")
	if logo == "" {
		// PNG 1×1 transparente.
		logo = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="
	}
	d := DadosCotacaoPDF{
		Codigo:   "COT 2026/0007",
		CriadoEm: time.Date(2026, 10, 20, 10, 0, 0, 0, time.Local),
		Loja: DadosRecibo{NegocioNome: "Ferragem Cossa", LojaNUIT: "400123456", LojaEndereco: "Av. Eduardo Mondlane 123, Matola",
			LojaTelefone: "84 123 4567", LojaEmail: "geral@cossa.co.mz"},
		Logo:            logo,
		ClienteNome:     "Construções Matola, Lda",
		ClienteNUIT:     "400555666",
		ClienteContacto: "84 111 2222 · compras@matola.co.mz",
		Itens: []ItemCotacaoPDF{
			{Descricao: "Cimento 50kg", NomeUnidade: "saco", Quantidade: 40, PrecoUnitario: 620, Desconto: 400, Subtotal: 24400},
			{Descricao: "Varão de ferro 12mm — 12 m", NomeUnidade: "unidade", Quantidade: 25, PrecoUnitario: 890, Subtotal: 22250},
		},
		Subtotal: 47050, DescontoTotal: 400, Total: 46650, TaxaIVA: 16, ValorIVA: 6434.48,
		Observacoes: "Entrega na obra incluída. Pagamento 50% na encomenda.",
	}
	pdf, err := GerarCotacaoPDF(d, time.Date(2026, 11, 4, 0, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("gerar: %v", err)
	}
	if n := len(regexp.MustCompile(`/Type /Page\b`).FindAll(pdf, -1)); n != 1 {
		t.Errorf("%d páginas, want 1", n)
	}
	if !bytes.Contains(pdf, []byte("/MediaBox [0 0 595.28 841.89]")) {
		t.Error("não é A4")
	}
	if !bytes.Contains(pdf, []byte("/Subtype /Image")) {
		t.Error("logótipo não foi incluído")
	}
	if dir := os.Getenv("COTACAO_DIR"); dir != "" {
		_ = os.WriteFile(dir+"/cotacao.pdf", pdf, 0o644)
	}
}
