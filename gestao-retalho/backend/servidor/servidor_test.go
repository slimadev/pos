package servidor

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/api"
)

func novoServidor(t *testing.T) (*httptest.Server, *api.App) {
	t.Helper()
	app, fechar, err := admin.AbrirLoja(filepath.Join(t.TempDir(), "loja.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(fechar)
	if _, err := app.OperadorService.Criar(context.Background(), "Lima", "1234"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(Novo(app, func() string { return "ABCD-EFGH" }).Handler())
	t.Cleanup(srv.Close)
	return srv, app
}

func chamar(t *testing.T, srv *httptest.Server, chave, aparelho, metodo string, args ...any) (resposta, int) {
	t.Helper()
	if args == nil {
		args = []any{}
	}
	a, _ := json.Marshal(args)
	corpo, _ := json.Marshal(pedido{Metodo: metodo, Args: a})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/chamar", bytes.NewReader(corpo))
	req.Header.Set("X-Chave", chave)
	req.Header.Set("X-Aparelho", aparelho)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var r resposta
	_ = json.NewDecoder(res.Body).Decode(&r)
	return r, res.StatusCode
}

func TestServidor_ChaveTurnoPorAparelhoEMetodos(t *testing.T) {
	srv, app := novoServidor(t)
	ops, _ := app.ListarOperadoresAtivos()

	if _, codigo := chamar(t, srv, "errada", "sunmi-1", "Ping"); codigo != http.StatusUnauthorized {
		t.Fatalf("chave errada: %d", codigo)
	}
	if r, _ := chamar(t, srv, "ABCD-EFGH", "sunmi-1", "IniciarTurno", ops[0].ID, "1234"); !r.OK {
		t.Fatalf("iniciar turno: %+v", r)
	}
	// O outro aparelho e o próprio PC continuam sem turno.
	r, _ := chamar(t, srv, "ABCD-EFGH", "sunmi-2", "ObterOperadorAtual")
	var op struct{ ID int64 }
	_ = json.Unmarshal(r.Dados, &op)
	if op.ID != 0 {
		t.Errorf("o sunmi-2 herdou o turno: %s", r.Dados)
	}
	if atual, _ := app.ObterOperadorAtual(); atual.ID != 0 {
		t.Errorf("o PC herdou o turno do aparelho: %+v", atual)
	}
	if r, _ := chamar(t, srv, "ABCD-EFGH", "sunmi-1", "AbrirMesa", "4", "", 2); !r.OK {
		t.Fatalf("abrir mesa no aparelho: %+v", r)
	}
	if r, _ := chamar(t, srv, "ABCD-EFGH", "sunmi-2", "AbrirMesa", "5", "", 2); r.OK {
		t.Fatal("o sunmi-2 não tem turno e abriu mesa")
	}
	mesas, _ := app.ListarMesas(false)
	if len(mesas) != 1 || mesas[0].GarcomNome != "Lima" {
		t.Fatalf("mesas no PC: %+v", mesas)
	}
	if r, _ := chamar(t, srv, "ABCD-EFGH", "sunmi-1", "FecharMesa", mesas[0].ContaID); r.OK || r.Erro == "" {
		t.Fatalf("fechar mesa no aparelho devia ser recusado: %+v", r)
	}
}
