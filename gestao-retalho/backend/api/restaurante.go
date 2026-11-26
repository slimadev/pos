package api

import (
	"fmt"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

// --- Restaurante: mesas, pedidos e contas ------------------------------------
//
// O garçom é sempre o operador do turno (IniciarTurno) — no aparelho do
// garçom e no caixa. Ver também o servidor para os aparelhos (backend/servidor).

// ListarMesas devolve as mesas abertas; soMinhas filtra as do operador do turno.
func (a *App) ListarMesas(soMinhas bool) ([]domain.MesaAberta, error) {
	if soMinhas {
		id := a.utilizadorAtual()
		return a.RestauranteService.ListarMesas(a.ctx, &id)
	}
	return a.RestauranteService.ListarMesas(a.ctx, nil)
}

func (a *App) ObterMesa(contaID int64) (usecase.MesaDetalhe, error) {
	return a.RestauranteService.ObterMesa(a.ctx, contaID)
}

// AbrirMesa: etiqueta é uma palavra do garçom para não confundir a conta
// ("camisa azul"); pode ir vazia.
func (a *App) AbrirMesa(mesa, etiqueta string, pessoas int64) (domain.MesaAberta, error) {
	if err := a.exigirTurno(); err != nil {
		return domain.MesaAberta{}, err
	}
	return a.RestauranteService.AbrirMesa(a.ctx, mesa, etiqueta, a.utilizadorAtual(), pessoas)
}

// NovaContaMesa abre mais uma conta na mesma mesa, para pagarem à parte.
func (a *App) NovaContaMesa(contaID int64, etiqueta string) (domain.MesaAberta, error) {
	if err := a.exigirTurno(); err != nil {
		return domain.MesaAberta{}, err
	}
	return a.RestauranteService.NovaConta(a.ctx, contaID, etiqueta, a.utilizadorAtual())
}

func (a *App) DefinirEtiquetaMesa(contaID int64, etiqueta string) error {
	return a.RestauranteService.DefinirEtiqueta(a.ctx, contaID, etiqueta)
}

func (a *App) LancarItemMesa(p usecase.AdicionarItemContaParams) (domain.ItemConta, error) {
	if err := a.exigirTurno(); err != nil {
		return domain.ItemConta{}, err
	}
	p.UtilizadorID = a.utilizadorAtual()
	return a.RestauranteService.LancarItem(a.ctx, p)
}

func (a *App) DefinirPessoasMesa(contaID, pessoas int64) error {
	return a.RestauranteService.DefinirPessoas(a.ctx, contaID, pessoas)
}

func (a *App) EnviarPedido(contaID int64) (usecase.ResultadoPedido, error) {
	if err := a.exigirTurno(); err != nil {
		return usecase.ResultadoPedido{}, err
	}
	return a.RestauranteService.EnviarPedido(a.ctx, contaID, a.utilizadorAtual())
}

func (a *App) ListarPedidosMesa(contaID int64) ([]domain.Pedido, error) {
	return a.RestauranteService.ListarPedidos(a.ctx, contaID)
}

// ReimprimirPedido: impressoraID nil = em todas as que o receberam.
func (a *App) ReimprimirPedido(pedidoID int64, impressoraID *int64) (usecase.ResultadoPedido, error) {
	return a.RestauranteService.ReimprimirPedido(a.ctx, pedidoID, impressoraID)
}

func (a *App) AnularItemMesa(p usecase.AnularItemParams) (usecase.ResultadoAnulacao, error) {
	if err := a.exigirTurno(); err != nil {
		return usecase.ResultadoAnulacao{}, err
	}
	p.GarcomID = a.utilizadorAtual()
	return a.RestauranteService.AnularItem(a.ctx, p)
}

func (a *App) DividirMesa(contaID int64, partes []usecase.ParteDivisao, etiqueta string) (domain.MesaAberta, error) {
	if err := a.exigirTurno(); err != nil {
		return domain.MesaAberta{}, err
	}
	return a.RestauranteService.DividirConta(a.ctx, contaID, partes, etiqueta)
}

func (a *App) TransferirMesa(contaID, novoGarcomID int64) error {
	if err := a.exigirTurno(); err != nil {
		return err
	}
	return a.RestauranteService.TransferirMesa(a.ctx, contaID, novoGarcomID)
}

func (a *App) JuntarMesas(origemID, destinoID int64) error {
	if err := a.exigirTurno(); err != nil {
		return err
	}
	return a.RestauranteService.JuntarMesas(a.ctx, origemID, destinoID, a.utilizadorAtual())
}

// ImprimirContaMesa imprime a conta para o cliente conferir. pessoas 0
// mantém o da mesa; impressoraID nil só devolve o texto (impressora do
// próprio aparelho).
func (a *App) ImprimirContaMesa(contaID, pessoas int64, impressoraID *int64) (usecase.ContaImpressa, error) {
	return a.RestauranteService.ImprimirConta(a.ctx, contaID, pessoas, impressoraID)
}

// DefinirClienteMesa: de quem é a conta (nil = anónimo). Só com cliente o
// fecho pode deixar o resto em dívida. Só no balcão (não é dos aparelhos).
func (a *App) DefinirClienteMesa(contaID int64, clienteID *int64) error {
	return a.RestauranteService.DefinirCliente(a.ctx, contaID, clienteID)
}

func (a *App) FecharMesa(contaID int64, p usecase.FecharContaParams) (FecharContaResponse, error) {
	if err := a.exigirTurno(); err != nil {
		return FecharContaResponse{}, err
	}
	p.UtilizadorID = a.utilizadorAtual()
	venda, alerta, err := a.RestauranteService.FecharMesa(a.ctx, contaID, p)
	if err != nil {
		return FecharContaResponse{}, err
	}
	return FecharContaResponse{Venda: venda, Alerta: alerta}, nil
}

// --- Impressoras ---------------------------------------------------------------

func (a *App) ListarImpressoras() ([]domain.Impressora, error) {
	return a.RestauranteService.ListarImpressoras(a.ctx)
}

func (a *App) GuardarImpressora(i domain.Impressora) (domain.Impressora, error) {
	return a.RestauranteService.GuardarImpressora(a.ctx, i)
}

func (a *App) TestarImpressora(id int64) error {
	return a.RestauranteService.TestarImpressora(a.ctx, id)
}

// RotaCategoria: os produtos da categoria saem nesta impressora de
// preparação (ImpressoraID nil = só no balcão).
type RotaCategoria struct {
	CategoriaID  int64
	ImpressoraID *int64
}

func (a *App) ListarRotasImpressao() ([]RotaCategoria, error) {
	m, err := a.RestauranteService.RotasImpressao(a.ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RotaCategoria, 0, len(m))
	for c, i := range m {
		imp := i
		out = append(out, RotaCategoria{CategoriaID: c, ImpressoraID: &imp})
	}
	return out, nil
}

func (a *App) DefinirRotaCategoria(categoriaID int64, impressoraID *int64) error {
	return a.RestauranteService.DefinirRotaCategoria(a.ctx, categoriaID, impressoraID)
}

// --- Caixa ---------------------------------------------------------------------

func (a *App) EstadoCaixa() (usecase.EstadoCaixa, error) {
	return a.CaixaService.Atual(a.ctx)
}

func (a *App) AbrirCaixa(fundo float64) (domain.SessaoCaixa, error) {
	if err := a.exigirTurno(); err != nil {
		return domain.SessaoCaixa{}, err
	}
	return a.CaixaService.Abrir(a.ctx, a.utilizadorAtual(), fundo)
}

// FecharCaixa recebe o contado por método (as chaves são os métodos:
// "dinheiro", "mpesa"…).
func (a *App) FecharCaixa(contagem map[string]float64, observacoes string) (usecase.FechoSessao, error) {
	if err := a.exigirTurno(); err != nil {
		return usecase.FechoSessao{}, err
	}
	c := make(map[domain.MetodoPagamento]float64, len(contagem))
	for m, v := range contagem {
		c[domain.MetodoPagamento(m)] = v
	}
	return a.CaixaService.Fechar(a.ctx, a.utilizadorAtual(), c, observacoes)
}

func (a *App) HistoricoCaixa(limite int64) ([]domain.SessaoCaixa, error) {
	return a.CaixaService.Historico(a.ctx, limite)
}

func (a *App) ObterFechoCaixa(sessaoID int64) (usecase.FechoSessao, error) {
	return a.CaixaService.ObterFecho(a.ctx, sessaoID)
}

// AutorizarGerente valida o PIN de um gerente (para o ecrã pedir antes de
// uma ação que precisa de autorização).
func (a *App) AutorizarGerente(pin string) (usecase.Operador, error) {
	return a.OperadorService.AutorizarGerente(a.ctx, pin)
}

// --- Servidor para os aparelhos dos garçons --------------------------------------

// AparelhoLigado: um aparelho que falou com o servidor deste PC.
type AparelhoLigado struct {
	ID       string
	Operador string // quem tem o turno aberto no aparelho ("" = ninguém)
	Visto    time.Time
}

type EstadoGestorServidor struct {
	Ligado    bool
	Erro      string
	Enderecos []string // IPs deste PC na rede local
	Aparelhos []AparelhoLigado
}

// GestorServidor é o servidor para os aparelhos, que só o PDV do desktop
// tem (backend/servidor) — o main liga-o em App.Servidor.
type GestorServidor interface {
	Aplicar(ativo bool, porta int) error
	Estado() EstadoGestorServidor
}

type EstadoServidor struct {
	// Disponivel: este PDV pode servir aparelhos (no próprio aparelho não).
	Disponivel bool
	Config     usecase.ServidorAparelhos
	EstadoGestorServidor
}

func (a *App) EstadoServidorAparelhos() (EstadoServidor, error) {
	cfg, err := a.ConfigService.ObterServidorAparelhos(a.ctx)
	if err != nil {
		return EstadoServidor{}, err
	}
	e := EstadoServidor{Config: cfg, Disponivel: a.Servidor != nil}
	if a.Servidor != nil {
		e.EstadoGestorServidor = a.Servidor.Estado()
	}
	return e, nil
}

func (a *App) DefinirServidorAparelhos(ativo bool, porta int) (EstadoServidor, error) {
	if a.Servidor == nil {
		return EstadoServidor{}, fmt.Errorf("só o computador do balcão pode servir os aparelhos")
	}
	if err := a.ConfigService.DefinirServidorAparelhos(a.ctx, ativo, porta); err != nil {
		return EstadoServidor{}, err
	}
	cfg, err := a.ConfigService.ObterServidorAparelhos(a.ctx)
	if err != nil {
		return EstadoServidor{}, err
	}
	aplicarErr := a.Servidor.Aplicar(cfg.Ativo, cfg.Porta)
	e, err := a.EstadoServidorAparelhos()
	if err != nil {
		return e, err
	}
	return e, aplicarErr
}

// NovaChaveServidor troca a chave da loja: os aparelhos têm de a escrever de novo.
func (a *App) NovaChaveServidor() (EstadoServidor, error) {
	if _, err := a.ConfigService.NovaChaveServidor(a.ctx); err != nil {
		return EstadoServidor{}, err
	}
	return a.EstadoServidorAparelhos()
}

// IniciarServidorAparelhos liga o servidor se a loja o tiver ativo — o main
// do PDV chama-o no arranque.
func (a *App) IniciarServidorAparelhos() error {
	if a.Servidor == nil {
		return nil
	}
	cfg, err := a.ConfigService.ObterServidorAparelhos(a.ctx)
	if err != nil {
		return err
	}
	return a.Servidor.Aplicar(cfg.Ativo, cfg.Porta)
}

// DefinirModoRestaurante liga/desliga as mesas (garçons, cozinha, caixa).
func (a *App) DefinirModoRestaurante(ativo bool) error {
	return a.ConfigService.DefinirModulo(a.ctx, domain.ModuloMesas, ativo)
}

func (a *App) DefinirGerente(operadorID int64, gerente bool) error {
	return a.OperadorService.DefinirGerente(a.ctx, operadorID, gerente)
}
