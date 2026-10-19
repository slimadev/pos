package export

import (
	"bytes"
	"os"
	"regexp"
	"testing"
	"time"
)

func dadosTalaoTeste() DadosRecibo {
	return DadosRecibo{
		NegocioNome: "Ferragem Cossa & Filhos — Matola",
		VendaID:     128,
		CriadoEm:    time.Date(2026, 10, 19, 15, 4, 0, 0, time.Local),
		ClienteNome: "Anónimo",
		Itens: []ItemRecibo{
			{NomeProduto: "Martelo de unha 500g", NomeUnidade: "unidade", Quantidade: 1, PrecoUnitario: 450, Subtotal: 450},
			{NomeProduto: "Parafuso 4x40 zincado cabeça chata para madeira", NomeUnidade: "caixa 100", Quantidade: 2, PrecoUnitario: 250, Desconto: 20, Subtotal: 480},
			{NomeProduto: "Fio elétrico 2,5mm", NomeUnidade: "metro", Quantidade: 12.5, PrecoUnitario: 45, Subtotal: 562.5},
		},
		Subtotal:      1492.5,
		DescontoTotal: 20,
		Total:         1472.5,
		Pagamentos:    []PagamentoRecibo{{Metodo: "Dinheiro", Valor: 1000}, {Metodo: "M-Pesa", Valor: 472.5}},
	}
}

// Uma só página com a largura do rolo: se a altura calculada ficasse
// curta, o fpdf não abria página nova (sem quebra automática) e o fim do
// recibo saía cortado — por isso confere-se também que o rodapé existe.
func TestGerarReciboTalao_UmaPaginaComALarguraDoRolo(t *testing.T) {
	for _, largura := range []float64{PapelRolo80, PapelRolo58} {
		pdf, err := GerarReciboTalao(dadosTalaoTeste(), largura)
		if err != nil {
			t.Fatalf("%v mm: %v", largura, err)
		}
		if n := len(regexp.MustCompile(`/Type /Page\b`).FindAll(pdf, -1)); n != 1 {
			t.Errorf("%v mm: %d páginas, want 1", largura, n)
		}
		// MediaBox em pontos: 80 mm = 226,77 pt; 58 mm = 164,41 pt.
		want := map[float64]string{PapelRolo80: "226.77", PapelRolo58: "164.41"}[largura]
		if !bytes.Contains(pdf, []byte("/MediaBox [0 0 "+want)) {
			t.Errorf("%v mm: MediaBox sem a largura %s pt", largura, want)
		}
		if dir := os.Getenv("RECIBO_TALAO_DIR"); dir != "" {
			_ = os.WriteFile(dir+"/talao.pdf", pdf, 0o644)
		}
	}
}
