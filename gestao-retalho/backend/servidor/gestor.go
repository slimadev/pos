package servidor

import (
	"context"
	"fmt"
	"sync"

	"gestao-retalho/backend/api"
)

// Gestor liga e desliga o servidor dentro do processo do PDV, conforme a
// configuração da loja (api.App.DefinirServidorAparelhos chama Aplicar).
type Gestor struct {
	app *api.App
	srv *Servidor

	mu    sync.Mutex
	parar func()
	porta int
	erro  string
}

func NovoGestor(app *api.App) *Gestor {
	chave := func() string {
		c, err := app.ConfigService.ObterServidorAparelhos(context.Background())
		if err != nil {
			return ""
		}
		return c.Chave
	}
	return &Gestor{app: app, srv: Novo(app, chave)}
}

// Aplicar põe o servidor no estado pedido (ligado na porta, ou desligado).
func (g *Gestor) Aplicar(ativo bool, porta int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.parar != nil && (!ativo || porta != g.porta) {
		g.parar()
		g.parar = nil
	}
	g.erro = ""
	if !ativo || g.parar != nil {
		return nil
	}
	parar, err := g.srv.Iniciar(porta)
	if err != nil {
		g.erro = fmt.Sprintf("não foi possível abrir a porta %d: %v", porta, err)
		return fmt.Errorf("%s", g.erro)
	}
	g.parar, g.porta = parar, porta
	return nil
}

func (g *Gestor) Estado() api.EstadoGestorServidor {
	g.mu.Lock()
	ligado, erro := g.parar != nil, g.erro
	g.mu.Unlock()
	return api.EstadoGestorServidor{Ligado: ligado, Erro: erro, Enderecos: EnderecosLocais(), Aparelhos: g.srv.Aparelhos()}
}

func (g *Gestor) Parar() {
	_ = g.Aplicar(false, 0)
}
