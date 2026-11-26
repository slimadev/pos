package gobridge

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/servidor"
)

func TestModoGarcom_ChamadasVaoParaOBalcao(t *testing.T) {
	app, fechar, err := admin.AbrirLoja(filepath.Join(t.TempDir(), "loja.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer fechar()
	op, err := app.OperadorService.Criar(context.Background(), "Lima", "1234")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(servidor.Novo(app, func() string { return "ABCD-EFGH" }).Handler())
	defer srv.Close()

	dir := t.TempDir()
	if err := carregarLigacao(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = gravarLigacao(nil) }()
	endereco := strings.TrimPrefix(srv.URL, "http://")

	if _, err := (ponte{}).LigarAoBalcao(endereco, "errada"); err == nil {
		t.Fatal("chave errada devia falhar")
	}
	est, err := (ponte{}).LigarAoBalcao(endereco, " abcd-efgh ")
	if err != nil || !est.ModoGarcom || !strings.HasPrefix(est.Aparelho, "aparelho-") {
		t.Fatalf("ligar: %+v, %v", est, err)
	}

	args, _ := json.Marshal([]any{op.ID, "1234"})
	if _, err := Chamar("api.App", "IniciarTurno", string(args)); err != nil {
		t.Fatal(err)
	}
	if _, err := Chamar("api.App", "AbrirMesa", `["12", "", 2]`); err != nil {
		t.Fatal(err)
	}
	mesas, _ := app.ListarMesas(false)
	if len(mesas) != 1 || mesas[0].Mesa != "12" {
		t.Fatalf("a mesa devia estar no balcão: %+v", mesas)
	}
	if _, err := Chamar("api.App", "CriarProduto", `[{}]`); err == nil || !strings.Contains(err.Error(), "balcão") {
		t.Fatalf("criar produto no aparelho: %v", err)
	}

	// A ligação fica gravada para o próximo arranque.
	ligacaoAtiv = nil
	if err := carregarLigacao(dir); err != nil || ligacaoAtual() == nil || ligacaoAtual().Endereco != endereco {
		t.Fatalf("ligação não ficou gravada: %+v, %v", ligacaoAtual(), err)
	}
	if est, _ := (ponte{}).DesligarDoBalcao(); est.ModoGarcom {
		t.Fatal("devia ter voltado ao modo local")
	}
}
