package gobridge

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/api"
	"gestao-retalho/backend/config"
)

// reiniciar repõe o estado global da ponte e aponta os dados para uma pasta
// temporária (cada teste é um terminal novo).
func reiniciar(t *testing.T) string {
	t.Helper()
	mu.Lock()
	adminApp, lojas, pdvApp = nil, nil, nil
	mu.Unlock()
	dir := t.TempDir()
	t.Cleanup(func() { config.DefinirDiretorioBase("") })
	if err := Iniciar(dir, "", nil); err != nil {
		t.Fatalf("Iniciar: %v", err)
	}
	return dir
}

// chamar faz o que a ponte JS faz: argumentos em JSON, resultado em JSON.
func chamar(t *testing.T, alvo, metodo string, saida any, args ...any) {
	t.Helper()
	j, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("args de %s: %v", metodo, err)
	}
	r, err := Chamar(alvo, metodo, string(j))
	if err != nil {
		t.Fatalf("%s.%s: %v", alvo, metodo, err)
	}
	if saida != nil {
		if err := json.Unmarshal([]byte(r), saida); err != nil {
			t.Fatalf("resultado de %s: %v (%s)", metodo, err, r)
		}
	}
}

const adminApp_ = "admin.AdminApp"
const pdv = "api.App"

// O percurso de um terminal novo, todo pela ponte: primeiro arranque no
// admin, login, PDV abre a loja criada, vende e o admin vê a venda.
func TestFluxoCompletoPelaPonte(t *testing.T) {
	reiniciar(t)

	var precisa bool
	chamar(t, adminApp_, "PrecisaConfigurarAdmin", &precisa)
	if !precisa {
		t.Fatal("terminal novo devia pedir configuração")
	}
	chamar(t, adminApp_, "ConfigurarPrimeiroArranque", nil, map[string]any{
		"NegocioNome": "Ferragem Teste", "TipoNegocio": "ferragem",
		"AdminNome": "Ana", "AdminUsername": "ana", "AdminPassword": "segredo123",
	})
	var sessao struct{ Nome string }
	chamar(t, adminApp_, "Login", &sessao, "ana", "segredo123")
	if sessao.Nome != "Ana" {
		t.Fatalf("sessão = %+v", sessao)
	}

	var configurado bool
	chamar(t, pdv, "EstaConfigurado", &configurado)
	if !configurado {
		t.Fatal("o PDV devia ver a loja criada pelo admin")
	}
	var armazens []struct {
		ID   int64
		Tipo string
	}
	chamar(t, pdv, "ListarArmazens", &armazens)
	if len(armazens) == 0 {
		t.Fatal("sem armazéns")
	}
	var catalogo []struct {
		Produto  struct{ ID int64 }
		Unidades []struct {
			ID         int64
			PrecoVenda float64
		}
	}
	chamar(t, pdv, "CatalogoPDV", &catalogo, armazens[0].ID)
	if len(catalogo) == 0 {
		t.Fatal("catálogo vazio (o seed da ferragem devia criar produtos)")
	}

	// Os produtos do seed começam sem stock: entrada pelo admin, como na loja.
	p := catalogo[0]
	chamar(t, adminApp_, "RegistarEntradaStock", nil, map[string]any{
		"ProdutoID": p.Produto.ID, "ArmazemDestinoID": armazens[0].ID, "Quantidade": 10, "CustoUnitario": 1,
	})
	// O PDV só vende com um operador identificado pelo PIN.
	var op struct{ ID int64 }
	chamar(t, adminApp_, "CriarOperador", &op, "Rui", "4321")
	chamar(t, pdv, "IniciarTurno", nil, op.ID, "4321")

	total := p.Unidades[0].PrecoVenda * 2
	var venda struct{ Venda struct{ ID int64 } }
	chamar(t, pdv, "RegistarVendaDireta", &venda, map[string]any{
		"ArmazemID":  armazens[0].ID,
		"Itens":      []map[string]any{{"ProdutoID": p.Produto.ID, "UnidadeProdutoID": p.Unidades[0].ID, "Quantidade": 2}},
		"Pagamentos": []map[string]any{{"Metodo": "dinheiro", "Valor": total}},
	})
	if venda.Venda.ID == 0 {
		t.Fatal("venda sem ID")
	}

	var recibo struct{ Total float64 }
	chamar(t, pdv, "ObterRecibo", &recibo, venda.Venda.ID)
	if recibo.Total != total {
		t.Errorf("recibo.Total = %v, want %v", recibo.Total, total)
	}

	hoje := time.Now().Format("2006-01-02")
	var vendas []struct{ Venda struct{ ID int64 } }
	chamar(t, adminApp_, "ListarVendas", &vendas, hoje, hoje)
	encontrou := false
	for _, v := range vendas {
		encontrou = encontrou || v.Venda.ID == venda.Venda.ID
	}
	if !encontrou {
		t.Errorf("o admin não vê a venda %d feita no PDV", venda.Venda.ID)
	}
}

// O arranque do terminal decide entre PDV e admin sem sessão iniciada.
func TestEstadoArranqueSemSessao(t *testing.T) {
	reiniciar(t)

	type estado struct {
		PrecisaConfigurar bool
		OperadoresAtivos  int
	}
	var e estado
	chamar(t, adminApp_, "ObterEstadoArranque", &e)
	if !e.PrecisaConfigurar || e.OperadoresAtivos != 0 {
		t.Fatalf("terminal novo: %+v", e)
	}

	chamar(t, adminApp_, "ConfigurarPrimeiroArranque", nil, map[string]any{
		"NegocioNome": "Ferragem Teste", "TipoNegocio": "ferragem",
		"AdminNome": "Ana", "AdminUsername": "ana", "AdminPassword": "segredo123",
	})
	chamar(t, adminApp_, "Logout", nil)
	chamar(t, adminApp_, "ObterEstadoArranque", &e)
	if e.PrecisaConfigurar || e.OperadoresAtivos != 0 {
		t.Fatalf("loja sem operadores: %+v", e)
	}

	chamar(t, adminApp_, "Login", nil, "ana", "segredo123")
	var op struct{ ID int64 }
	chamar(t, adminApp_, "CriarOperador", &op, "Rui", "4321")
	chamar(t, adminApp_, "Logout", nil)
	chamar(t, adminApp_, "ObterEstadoArranque", &e)
	if e.OperadoresAtivos != 1 {
		t.Fatalf("com um operador: %+v", e)
	}

	chamar(t, adminApp_, "Login", nil, "ana", "segredo123")
	chamar(t, adminApp_, "DesativarOperador", nil, op.ID)
	chamar(t, adminApp_, "ObterEstadoArranque", &e)
	if e.OperadoresAtivos != 0 {
		t.Fatalf("operador desativado ainda conta: %+v", e)
	}
}

func TestFusoHorarioDoTerminal(t *testing.T) {
	original := time.Local
	t.Cleanup(func() { time.Local = original })
	mu.Lock()
	adminApp, lojas, pdvApp = nil, nil, nil
	mu.Unlock()
	t.Cleanup(func() { config.DefinirDiretorioBase("") })

	if err := Iniciar(t.TempDir(), "Africa/Maputo", nil); err != nil {
		t.Fatalf("Iniciar: %v", err)
	}
	if time.Local.String() != "Africa/Maputo" {
		t.Errorf("time.Local = %s, want Africa/Maputo", time.Local)
	}

	mu.Lock()
	adminApp = nil
	mu.Unlock()
	if err := Iniciar(t.TempDir(), "Lua/Inventado", nil); err == nil {
		t.Error("fuso inválido devia falhar")
	}
}

func TestErrosChegamComoErro(t *testing.T) {
	reiniciar(t)

	casos := []struct {
		nome, alvo, metodo, args, contem string
	}{
		{"método inexistente", adminApp_, "NaoExiste", "[]", "desconhecido"},
		{"método interno", adminApp_, "OnStartup", "[]", "não disponível"},
		{"não exportado", adminApp_, "exigirSessao", "[]", "não disponível"},
		{"alvo inexistente", "x.Y", "Login", "[]", "alvo"},
		{"argumentos a mais", adminApp_, "Logout", `[1]`, "argumento"},
		{"tipo errado", adminApp_, "Login", `[1, 2]`, "argumento 1"},
		{"sem loja ainda", pdv, "ListarArmazens", "[]", "nenhuma loja"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			_, err := Chamar(c.alvo, c.metodo, c.args)
			if err == nil || !strings.Contains(err.Error(), c.contem) {
				t.Errorf("erro = %v, want contendo %q", err, c.contem)
			}
		})
	}

	// Erro do próprio serviço (password errada) chega com a mensagem dele.
	chamar(t, adminApp_, "ConfigurarPrimeiroArranque", nil, map[string]any{
		"NegocioNome": "Loja", "TipoNegocio": "ferragem",
		"AdminNome": "Ana", "AdminUsername": "ana", "AdminPassword": "segredo123",
	})
	if _, err := Chamar(adminApp_, "Login", `["ana","errada"]`); err == nil {
		t.Error("login com password errada devia falhar")
	}
}

// Num terminal novo (sem administrador) restaura-se sem login — é assim que
// a loja volta depois de uma avaria; com administrador, só com sessão.
func TestRestaurarSemLoginSoNumTerminalNovo(t *testing.T) {
	reiniciar(t)
	semSessao := func() error {
		_, err := Chamar(adminApp_, "PrepararRestauroCopia", `["/nao/existe.zip"]`)
		return err
	}
	if err := semSessao(); err == nil || strings.Contains(err.Error(), "sessão") {
		t.Errorf("terminal novo: erro = %v, devia falhar só por o ficheiro não existir", err)
	}
	chamar(t, adminApp_, "ConfigurarPrimeiroArranque", nil, map[string]any{
		"NegocioNome": "Loja", "TipoNegocio": "ferragem",
		"AdminNome": "Ana", "AdminUsername": "ana", "AdminPassword": "segredo123",
	})
	chamar(t, adminApp_, "Logout", nil)
	if err := semSessao(); err == nil || !strings.Contains(err.Error(), "sessão") {
		t.Errorf("com administrador e sem login: erro = %v, devia pedir sessão", err)
	}
}

// Todos os métodos que o frontend pode chamar têm de passar por JSON: sem
// canais, funções nem variádicos nos parâmetros ou no resultado.
func TestTodosOsMetodosSaoChamaveisPelaPonte(t *testing.T) {
	for _, tipo := range []reflect.Type{reflect.TypeOf(&admin.AdminApp{}), reflect.TypeOf(&api.App{})} {
		for i := 0; i < tipo.NumMethod(); i++ {
			m := tipo.Method(i)
			if m.Name == "OnStartup" {
				continue
			}
			ft := m.Type
			if ft.IsVariadic() {
				t.Errorf("%s.%s é variádico", tipo, m.Name)
			}
			for j := 1; j < ft.NumIn(); j++ {
				if !serializavel(ft.In(j)) {
					t.Errorf("%s.%s: parâmetro %d (%s) não passa por JSON", tipo, m.Name, j, ft.In(j))
				}
			}
			if ft.NumOut() > 0 && ft.Out(0) != tipoErro && !serializavel(ft.Out(0)) {
				t.Errorf("%s.%s: resultado (%s) não passa por JSON", tipo, m.Name, ft.Out(0))
			}
		}
	}
}

func serializavel(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Chan, reflect.Func, reflect.UnsafePointer, reflect.Complex64, reflect.Complex128:
		return false
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return serializavel(t.Elem())
	case reflect.Map:
		return serializavel(t.Key()) && serializavel(t.Elem())
	}
	return true
}
