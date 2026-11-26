package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/export"
	"gestao-retalho/backend/impressao"
	"gestao-retalho/backend/repository"
)

const eps = 1e-9

var (
	ErrMesaNaoEncontrada = errors.New("mesa não encontrada ou já fechada")
	ErrNadaPorEnviar     = errors.New("não há nada novo para enviar nesta mesa")
	ErrCaixaFechado      = errors.New("o caixa está fechado — abra o caixa antes de fechar contas")
)

// RestauranteService é o serviço das mesas: o garçom abre a mesa, lança o
// que pedem e envia o pedido — sai logo impresso no balcão (tudo) e na
// cozinha (só o que é de lá, pela categoria do produto). O fecho é sempre no
// caixa. A conta por baixo é a mesma do bar (ContaService): o stock e os
// ingredientes dos produtos compostos só saem ao fechar.
type RestauranteService struct {
	rest         *repository.RestauranteRepository
	vendas       *repository.VendaRepository
	contas       *ContaService
	operadores   *OperadorService
	utilizadores *repository.UtilizadorRepository
	negocio      *repository.NegocioRepository
	configs      *repository.ConfigRepository
	impressor    impressao.Impressor
	clientes     *repository.ClienteRepository

	// Um envio de cada vez: dois aparelhos a enviar a mesma mesa ao mesmo
	// tempo não podem imprimir os mesmos itens duas vezes.
	mu sync.Mutex
}

func NewRestauranteService(
	rest *repository.RestauranteRepository,
	vendas *repository.VendaRepository,
	contas *ContaService,
	operadores *OperadorService,
	utilizadores *repository.UtilizadorRepository,
	negocio *repository.NegocioRepository,
	configs *repository.ConfigRepository,
	impressor impressao.Impressor,
	clientes *repository.ClienteRepository,
) *RestauranteService {
	return &RestauranteService{
		rest: rest, vendas: vendas, contas: contas, operadores: operadores, utilizadores: utilizadores,
		negocio: negocio, configs: configs, impressor: impressor, clientes: clientes,
	}
}

// --- Quem pode mexer em quê ---------------------------------------------------
//
// Cada garçom só mexe nas suas mesas. O caixa e o gerente mexem nas de
// todos (anular, dividir, passar a outro garçom, juntar…) mas não lançam
// pedidos nas mesas dos outros. Fechar contas, vender direto e abrir/fechar
// o caixa é só do caixa e do gerente.

type AcaoMesa int

const (
	// AcaoLancar: lançar itens — só o garçom da mesa.
	AcaoLancar AcaoMesa = iota
	// AcaoGerir: enviar, anular, dividir, passar, juntar, pessoas, conta…
	// — o garçom da mesa, o caixa ou o gerente.
	AcaoGerir
	// AcaoFechar: receber a conta (e escolher o cliente do fiado) — só o
	// caixa ou o gerente.
	AcaoFechar
)

// ErrSoCaixa: a ação é do caixa ou do gerente.
var ErrSoCaixa = errors.New("só o caixa ou o gerente pode fazer isto — quem é caixa marca-se na Administração, em Utilizadores")

// ExigirCaixa: o operador é caixa ou gerente.
func (s *RestauranteService) ExigirCaixa(ctx context.Context, utilizadorID int64) error {
	u, err := s.utilizadores.Obter(ctx, utilizadorID)
	if err != nil {
		return err
	}
	if !u.PodeGerirMesas() {
		return ErrSoCaixa
	}
	return nil
}

// ExigirPermissao verifica se o operador pode fazer a ação nesta mesa.
func (s *RestauranteService) ExigirPermissao(ctx context.Context, contaID, utilizadorID int64, acao AcaoMesa) error {
	m, err := s.obterMesa(ctx, contaID)
	if err != nil {
		return err
	}
	return s.exigirPermissao(ctx, m, utilizadorID, acao)
}

func (s *RestauranteService) exigirPermissao(ctx context.Context, m domain.MesaAberta, utilizadorID int64, acao AcaoMesa) error {
	u, err := s.utilizadores.Obter(ctx, utilizadorID)
	if err != nil {
		return err
	}
	dono := m.GarcomID == utilizadorID
	switch acao {
	case AcaoLancar:
		if !dono {
			return fmt.Errorf("%s é de %s — só quem tem a mesa lança pedidos nela", rotuloMesa(m.Mesa), m.GarcomNome)
		}
	case AcaoGerir:
		if !dono && !u.PodeGerirMesas() {
			return fmt.Errorf("%s é de %s — só quem tem a mesa, o caixa ou o gerente podem mexer nela", rotuloMesa(m.Mesa), m.GarcomNome)
		}
	case AcaoFechar:
		if !u.PodeGerirMesas() {
			return ErrSoCaixa
		}
	}
	return nil
}

// --- Mesas -------------------------------------------------------------------

// ListarMesas devolve as mesas abertas; com garcomID, só as desse garçom.
func (s *RestauranteService) ListarMesas(ctx context.Context, garcomID *int64) ([]domain.MesaAberta, error) {
	todas, err := s.rest.ListarMesasAbertas(ctx, domain.NegocioID)
	if err != nil || garcomID == nil {
		return todas, err
	}
	out := make([]domain.MesaAberta, 0, len(todas))
	for _, m := range todas {
		if m.GarcomID == *garcomID {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *RestauranteService) obterMesa(ctx context.Context, contaID int64) (domain.MesaAberta, error) {
	todas, err := s.rest.ListarMesasAbertas(ctx, domain.NegocioID)
	if err != nil {
		return domain.MesaAberta{}, err
	}
	for _, m := range todas {
		if m.ContaID == contaID {
			return m, nil
		}
	}
	return domain.MesaAberta{}, ErrMesaNaoEncontrada
}

type MesaDetalhe struct {
	Mesa  domain.MesaAberta
	Itens []domain.ItemMesa
}

func (s *RestauranteService) ObterMesa(ctx context.Context, contaID int64) (MesaDetalhe, error) {
	m, err := s.obterMesa(ctx, contaID)
	if err != nil {
		return MesaDetalhe{}, err
	}
	itens, err := s.rest.ListarItens(ctx, contaID)
	if err != nil {
		return MesaDetalhe{}, err
	}
	return MesaDetalhe{Mesa: m, Itens: itens}, nil
}

// rotuloMesa: "5" vira "Mesa 5", "5 (2)" vira "Mesa 5 (2)"; "Varanda 2"
// fica como está.
func rotuloMesa(mesa string) string {
	base := baseMesa(mesa)
	if base == "" {
		return mesa
	}
	for _, r := range base {
		if r < '0' || r > '9' {
			return mesa
		}
	}
	return "Mesa " + mesa
}

var sufixoConta = regexp.MustCompile(`\s*\(\d+\)$`)

// baseMesa tira o número da conta: "5 (2)" é a mesa "5".
func baseMesa(mesa string) string {
	return sufixoConta.ReplaceAllString(mesa, "")
}

// MaxEtiqueta: o bastante para "camisa azul" ou "aniversariante", e cabe
// numa linha do talão.
const MaxEtiqueta = 30

func limparEtiqueta(e string) (string, error) {
	e = strings.Join(strings.Fields(e), " ")
	if utf8.RuneCountInString(e) > MaxEtiqueta {
		return "", fmt.Errorf("a etiqueta tem de ter até %d letras", MaxEtiqueta)
	}
	return e, nil
}

// nomeConta: "Mesa 5 (2) · camisa azul" — como a conta aparece nos talões.
func nomeConta(m domain.MesaAberta) string {
	if m.Etiqueta == "" {
		return rotuloMesa(m.Mesa)
	}
	return rotuloMesa(m.Mesa) + " · " + m.Etiqueta
}

// proximaConta escolhe o nome da próxima conta da mesa: "5 (2)", "5 (3)"…
func proximaConta(abertas []domain.MesaAberta, mesa string) string {
	base := baseMesa(mesa)
	for n := 2; ; n++ {
		candidato := fmt.Sprintf("%s (%d)", base, n)
		livre := true
		for _, a := range abertas {
			if mesmoNome(a.Mesa, candidato) {
				livre = false
			}
		}
		if livre {
			return candidato
		}
	}
}

func (s *RestauranteService) garcomAtivo(ctx context.Context, id int64) (domain.Utilizador, error) {
	u, err := s.operadores.obterOperador(ctx, id)
	if err != nil {
		return domain.Utilizador{}, err
	}
	if !u.Ativo {
		return domain.Utilizador{}, ErrOperadorInativo
	}
	return u, nil
}

// AbrirMesa abre a conta de uma mesa para um garçom. Uma mesa já aberta
// não se abre outra vez — junta-se o pedido à que lá está.
func (s *RestauranteService) AbrirMesa(ctx context.Context, mesa, etiqueta string, garcomID, pessoas int64) (domain.MesaAberta, error) {
	mesa = strings.Join(strings.Fields(mesa), " ")
	if mesa == "" {
		return domain.MesaAberta{}, fmt.Errorf("diga o número ou o nome da mesa")
	}
	etiqueta, err := limparEtiqueta(etiqueta)
	if err != nil {
		return domain.MesaAberta{}, err
	}
	if pessoas < 1 {
		pessoas = 1
	}
	if _, err := s.garcomAtivo(ctx, garcomID); err != nil {
		return domain.MesaAberta{}, err
	}
	abertas, err := s.rest.ListarMesasAbertas(ctx, domain.NegocioID)
	if err != nil {
		return domain.MesaAberta{}, err
	}
	for _, m := range abertas {
		if mesmoNome(m.Mesa, mesa) {
			return domain.MesaAberta{}, fmt.Errorf("%s já está aberta (garçom %s) — para contas separadas, abra-a e use «Nova conta»", rotuloMesa(m.Mesa), m.GarcomNome)
		}
	}
	var contaID int64
	err = s.rest.WithTxVendas(ctx, s.vendas, func(rest *repository.RestauranteRepository, vendas *repository.VendaRepository) error {
		c, err := vendas.CriarConta(ctx, domain.NegocioID, domain.ClienteAnonimoID, rotuloMesa(mesa), garcomID)
		if err != nil {
			return err
		}
		contaID = c.ID
		if err := rest.DefinirContaMesa(ctx, c.ID, mesa, garcomID, pessoas); err != nil {
			return err
		}
		return rest.DefinirEtiquetaMesa(ctx, c.ID, etiqueta)
	})
	if err != nil {
		return domain.MesaAberta{}, err
	}
	return s.obterMesa(ctx, contaID)
}

// LancarItem acrescenta um produto à mesa — ainda não sai na cozinha: só
// ao EnviarPedido, para o garçom poder juntar o pedido todo da mesa.
func (s *RestauranteService) LancarItem(ctx context.Context, p AdicionarItemContaParams) (domain.ItemConta, error) {
	if _, err := s.obterMesa(ctx, p.ContaID); err != nil {
		return domain.ItemConta{}, err
	}
	return s.contas.AdicionarItem(ctx, p)
}

func (s *RestauranteService) DefinirPessoas(ctx context.Context, contaID, pessoas int64) error {
	if pessoas < 1 {
		return fmt.Errorf("o número de pessoas tem de ser pelo menos 1")
	}
	m, err := s.obterMesa(ctx, contaID)
	if err != nil {
		return err
	}
	return s.rest.DefinirContaMesa(ctx, contaID, m.Mesa, m.GarcomID, pessoas)
}

// NovaConta abre mais uma conta na mesma mesa ("5 (2)"), para quem quer
// pagar à parte desde o início. Fica com o garçom que a abre.
func (s *RestauranteService) NovaConta(ctx context.Context, contaID int64, etiqueta string, garcomID int64) (domain.MesaAberta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	etiqueta, err := limparEtiqueta(etiqueta)
	if err != nil {
		return domain.MesaAberta{}, err
	}
	m, err := s.obterMesa(ctx, contaID)
	if err != nil {
		return domain.MesaAberta{}, err
	}
	if _, err := s.garcomAtivo(ctx, garcomID); err != nil {
		return domain.MesaAberta{}, err
	}
	if err := s.exigirPermissao(ctx, m, garcomID, AcaoGerir); err != nil {
		return domain.MesaAberta{}, err
	}
	abertas, err := s.rest.ListarMesasAbertas(ctx, domain.NegocioID)
	if err != nil {
		return domain.MesaAberta{}, err
	}
	nome := proximaConta(abertas, m.Mesa)
	var novaID int64
	err = s.rest.WithTxVendas(ctx, s.vendas, func(rest *repository.RestauranteRepository, vendas *repository.VendaRepository) error {
		c, err := vendas.CriarConta(ctx, domain.NegocioID, domain.ClienteAnonimoID, rotuloMesa(nome), garcomID)
		if err != nil {
			return err
		}
		novaID = c.ID
		if err := rest.DefinirContaMesa(ctx, c.ID, nome, garcomID, 1); err != nil {
			return err
		}
		return rest.DefinirEtiquetaMesa(ctx, c.ID, etiqueta)
	})
	if err != nil {
		return domain.MesaAberta{}, err
	}
	return s.obterMesa(ctx, novaID)
}

// DefinirEtiqueta muda (ou apaga, com "") a etiqueta da conta.
func (s *RestauranteService) DefinirEtiqueta(ctx context.Context, contaID int64, etiqueta string) error {
	etiqueta, err := limparEtiqueta(etiqueta)
	if err != nil {
		return err
	}
	if _, err := s.obterMesa(ctx, contaID); err != nil {
		return err
	}
	return s.rest.DefinirEtiquetaMesa(ctx, contaID, etiqueta)
}

// DefinirCliente diz de quem é a conta da mesa — só com um cliente
// identificado o fecho pode deixar o resto em dívida (fiado). nil volta ao
// cliente anónimo.
func (s *RestauranteService) DefinirCliente(ctx context.Context, contaID int64, clienteID *int64) error {
	if _, err := s.obterMesa(ctx, contaID); err != nil {
		return err
	}
	id := domain.ClienteAnonimoID
	if clienteID != nil && *clienteID != domain.ClienteAnonimoID {
		c, err := s.clientes.Obter(ctx, *clienteID)
		if err != nil {
			return fmt.Errorf("cliente não encontrado")
		}
		if !c.Ativo {
			return fmt.Errorf("o cliente %s está inativo", c.Nome)
		}
		id = c.ID
	}
	return s.vendas.DefinirClienteConta(ctx, contaID, id)
}

// TransferirMesa passa a mesa (com tudo o que já tem) para outro garçom.
func (s *RestauranteService) TransferirMesa(ctx context.Context, contaID, novoGarcomID int64) error {
	m, err := s.obterMesa(ctx, contaID)
	if err != nil {
		return err
	}
	if m.GarcomID == novoGarcomID {
		return fmt.Errorf("a mesa já é deste garçom")
	}
	if _, err := s.garcomAtivo(ctx, novoGarcomID); err != nil {
		return err
	}
	return s.rest.DefinirContaMesa(ctx, contaID, m.Mesa, novoGarcomID, m.Pessoas)
}

// JuntarMesas passa os itens da mesa de origem para a de destino (o que já
// foi enviado continua enviado) e fecha a origem como cancelada, com o motivo.
func (s *RestauranteService) JuntarMesas(ctx context.Context, origemID, destinoID, utilizadorID int64) error {
	if origemID == destinoID {
		return fmt.Errorf("escolha duas mesas diferentes")
	}
	origem, err := s.obterMesa(ctx, origemID)
	if err != nil {
		return err
	}
	destino, err := s.obterMesa(ctx, destinoID)
	if err != nil {
		return err
	}
	for _, m := range []domain.MesaAberta{origem, destino} {
		if err := s.exigirPermissao(ctx, m, utilizadorID, AcaoGerir); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rest.WithTxVendas(ctx, s.vendas, func(rest *repository.RestauranteRepository, vendas *repository.VendaRepository) error {
		itens, err := vendas.ListarItensConta(ctx, origemID)
		if err != nil {
			return err
		}
		for _, i := range itens {
			if err := vendas.TransferirItemConta(ctx, i.ID, destinoID); err != nil {
				return err
			}
		}
		if err := rest.DefinirContaMesa(ctx, destinoID, destino.Mesa, destino.GarcomID, origem.Pessoas+destino.Pessoas); err != nil {
			return err
		}
		_, err = vendas.CancelarConta(ctx, origemID, utilizadorID, "Junta à "+rotuloMesa(destino.Mesa))
		return err
	})
}

// --- Pedidos -------------------------------------------------------------------

type ResultadoImpressao struct {
	ImpressoraID int64
	Nome         string
	Erro         string // vazio = impresso
}

type ResultadoPedido struct {
	Pedido     domain.Pedido
	Impressoes []ResultadoImpressao
	// Texto é o talão a 32 colunas, para a impressora do próprio aparelho.
	Texto string
}

type rotas struct {
	impressoras  []domain.Impressora // só as ativas
	porCategoria map[int64]int64     // categoria → impressora de preparação ativa
}

func (s *RestauranteService) carregarRotas(ctx context.Context) (rotas, error) {
	todas, err := s.rest.ListarImpressoras(ctx)
	if err != nil {
		return rotas{}, err
	}
	mapa, err := s.rest.ImpressoraPorCategoria(ctx)
	if err != nil {
		return rotas{}, err
	}
	r := rotas{porCategoria: map[int64]int64{}}
	prep := map[int64]bool{}
	for _, i := range todas {
		if !i.Ativa {
			continue
		}
		r.impressoras = append(r.impressoras, i)
		if i.Funcao == domain.ImpressoraPreparacao {
			prep[i.ID] = true
		}
	}
	for cat, imp := range mapa {
		if prep[imp] {
			r.porCategoria[cat] = imp
		}
	}
	return r, nil
}

func (r rotas) destino(categoriaID *int64) *int64 {
	if categoriaID == nil {
		return nil
	}
	if id, ok := r.porCategoria[*categoriaID]; ok {
		return &id
	}
	return nil
}

func (s *RestauranteService) nomeUtilizador(ctx context.Context, id int64) string {
	u, err := s.utilizadores.Obter(ctx, id)
	if err != nil {
		return "?"
	}
	return u.Nome
}

// EnviarPedido manda para a impressão o que foi lançado na mesa e ainda
// não tinha sido enviado. O balcão recebe o pedido todo; cada impressora de
// preparação (cozinha…) só as linhas das suas categorias. Os itens ficam
// marcados como enviados mesmo que uma impressora falhe — o resultado diz
// qual falhou, para reimprimir ou imprimir no aparelho.
func (s *RestauranteService) EnviarPedido(ctx context.Context, contaID, garcomID int64) (ResultadoPedido, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, err := s.obterMesa(ctx, contaID)
	if err != nil {
		return ResultadoPedido{}, err
	}
	itens, err := s.rest.ListarItens(ctx, contaID)
	if err != nil {
		return ResultadoPedido{}, err
	}
	r, err := s.carregarRotas(ctx)
	if err != nil {
		return ResultadoPedido{}, err
	}
	var linhas []domain.LinhaPedido
	var enviados []domain.ItemMesa
	for _, i := range itens {
		q := i.Quantidade - i.Enviada
		if q <= eps {
			continue
		}
		linhas = append(linhas, domain.LinhaPedido{
			ProdutoID: i.ProdutoID, Descricao: i.Produto, Unidade: i.Unidade, Quantidade: q, ImpressoraID: r.destino(i.CategoriaID),
		})
		enviados = append(enviados, i)
	}
	if len(linhas) == 0 {
		return ResultadoPedido{}, ErrNadaPorEnviar
	}

	var pedido domain.Pedido
	err = s.rest.WithTx(ctx, func(tx *repository.RestauranteRepository) error {
		var err error
		pedido, err = tx.CriarPedido(ctx, domain.Pedido{ContaID: contaID, Tipo: domain.TipoPedidoEnvio, Mesa: nomeConta(m), GarcomID: garcomID, Linhas: linhas})
		if err != nil {
			return err
		}
		for _, i := range enviados {
			if err := tx.DefinirQuantidadeEnviada(ctx, i.ID, i.Quantidade); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ResultadoPedido{}, err
	}
	return s.imprimirPedido(ctx, pedido, r, nil, false), nil
}

// ReimprimirPedido volta a imprimir um pedido (2.ª via) — em todas as
// impressoras a que foi, ou só numa.
func (s *RestauranteService) ReimprimirPedido(ctx context.Context, pedidoID int64, impressoraID *int64) (ResultadoPedido, error) {
	p, err := s.rest.ObterPedido(ctx, pedidoID)
	if err != nil {
		return ResultadoPedido{}, err
	}
	r, err := s.carregarRotas(ctx)
	if err != nil {
		return ResultadoPedido{}, err
	}
	return s.imprimirPedido(ctx, p, r, impressoraID, true), nil
}

func (s *RestauranteService) ListarPedidos(ctx context.Context, contaID int64) ([]domain.Pedido, error) {
	return s.rest.ListarPedidosDaConta(ctx, contaID)
}

func (s *RestauranteService) imprimirPedido(ctx context.Context, p domain.Pedido, r rotas, so *int64, segundaVia bool) ResultadoPedido {
	garcom := s.nomeUtilizador(ctx, p.GarcomID)
	var autorizou string
	if p.AutorizadoPor != nil {
		autorizou = s.nomeUtilizador(ctx, *p.AutorizadoPor)
	}
	res := ResultadoPedido{Pedido: p, Impressoes: []ResultadoImpressao{}}
	res.Texto = talaoPedido(p, p.Linhas, garcom, autorizou, 32, segundaVia).TextoSimples()
	for _, imp := range r.impressoras {
		if so != nil && imp.ID != *so {
			continue
		}
		linhas := p.Linhas
		if imp.Funcao != domain.ImpressoraBalcao {
			linhas = nil
			for _, l := range p.Linhas {
				if l.ImpressoraID != nil && *l.ImpressoraID == imp.ID {
					linhas = append(linhas, l)
				}
			}
		}
		if len(linhas) == 0 {
			continue
		}
		t := talaoPedido(p, linhas, garcom, autorizou, int(imp.Colunas), segundaVia)
		ri := ResultadoImpressao{ImpressoraID: imp.ID, Nome: imp.Nome}
		if err := s.impressor.Enviar(ctx, imp.Endereco, int(imp.Porta), t.ESCPOS()); err != nil {
			ri.Erro = err.Error()
		}
		res.Impressoes = append(res.Impressoes, ri)
	}
	return res
}

func descricaoLinha(l domain.LinhaPedido) string {
	d := l.Descricao
	if u := strings.TrimSpace(l.Unidade); u != "" && !mesmoNome(u, "unidade") && !mesmoNome(u, "un") && !mesmoNome(u, "und") {
		d += " (" + u + ")"
	}
	return qtd(l.Quantidade) + " x " + d
}

func talaoPedido(p domain.Pedido, linhas []domain.LinhaPedido, garcom, autorizou string, colunas int, segundaVia bool) *impressao.Talao {
	t := impressao.NovoTalao(colunas)
	if p.Tipo == domain.TipoPedidoAnulacao {
		t.Titulo("*** ANULADO ***")
	}
	t.Titulo(strings.ToUpper(rotuloMesa(p.Mesa)))
	t.Centro(fmt.Sprintf("Pedido #%d · %s", p.ID, p.CriadoEm.Local().Format("02/01 15:04")))
	if segundaVia {
		t.Centro("*** 2.ª VIA ***")
	}
	if p.Tipo == domain.TipoPedidoAnulacao {
		t.Texto("Garçom " + garcom + " anulou:")
	} else {
		t.Texto("Garçom " + garcom + " pediu:")
	}
	t.Separador()
	for _, l := range linhas {
		t.Grande(descricaoLinha(l))
	}
	t.Separador()
	if p.Tipo == domain.TipoPedidoAnulacao {
		if p.Motivo != nil {
			t.Texto("Motivo: " + *p.Motivo)
		}
		if autorizou != "" {
			t.Texto("Autorizado por " + autorizou)
		}
	}
	return t
}

// --- Anular ----------------------------------------------------------------------

type AnularItemParams struct {
	ItemID     int64
	Quantidade float64
	GarcomID   int64
	// Só para o que já foi enviado para preparação: o PIN de um gerente e
	// o motivo. O que ainda não foi enviado tira-se sem autorização.
	PINGerente string
	Motivo     string
}

type ResultadoAnulacao struct {
	// Pedido de anulação impresso (nil se só saiu o que não fora enviado).
	Anulacao *ResultadoPedido
	// PrecisaAutorizacao: a quantidade inclui itens já enviados e faltou o
	// PIN do gerente ou o motivo — nada foi alterado.
	PrecisaAutorizacao bool
}

// AnularItem tira uma quantidade de uma linha da mesa. Primeiro sai o que
// ainda não foi enviado; se tocar no que já foi para a cozinha, pede o PIN
// de um gerente e o motivo, e imprime "ANULADO" onde o pedido saiu.
func (s *RestauranteService) AnularItem(ctx context.Context, p AnularItemParams) (ResultadoAnulacao, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, err := s.vendas.ObterItemConta(ctx, p.ItemID)
	if err != nil {
		return ResultadoAnulacao{}, err
	}
	m, err := s.obterMesa(ctx, item.ContaID)
	if err != nil {
		return ResultadoAnulacao{}, err
	}
	if err := s.exigirPermissao(ctx, m, p.GarcomID, AcaoGerir); err != nil {
		return ResultadoAnulacao{}, err
	}
	quem, err := s.utilizadores.Obter(ctx, p.GarcomID)
	if err != nil {
		return ResultadoAnulacao{}, err
	}
	itens, err := s.rest.ListarItens(ctx, item.ContaID)
	if err != nil {
		return ResultadoAnulacao{}, err
	}
	var im domain.ItemMesa
	for _, i := range itens {
		if i.ID == p.ItemID {
			im = i
		}
	}
	if p.Quantidade <= eps || p.Quantidade > im.Quantidade+eps {
		return ResultadoAnulacao{}, fmt.Errorf("quantidade a anular tem de estar entre 0 e %s", qtd(im.Quantidade))
	}
	porEnviar := math.Max(0, im.Quantidade-im.Enviada)
	anularEnviada := math.Max(0, p.Quantidade-porEnviar)
	enviada := math.Min(im.Enviada, im.Quantidade)

	var autorizador *Operador
	motivo := strings.TrimSpace(p.Motivo)
	if anularEnviada > eps {
		// O caixa e o gerente autorizam-se a si próprios; o garçom chama-os.
		autoriza := quem.PodeGerirMesas() && quem.Role != domain.RoleAdmin
		if motivo == "" || (!autoriza && p.PINGerente == "") {
			return ResultadoAnulacao{PrecisaAutorizacao: true}, nil
		}
		if autoriza {
			o := operadorDe(quem)
			autorizador = &o
		} else {
			g, err := s.operadores.AutorizarGerente(ctx, p.PINGerente)
			if err != nil {
				return ResultadoAnulacao{}, err
			}
			autorizador = &g
		}
	}

	r, err := s.carregarRotas(ctx)
	if err != nil {
		return ResultadoAnulacao{}, err
	}
	nova := im.Quantidade - p.Quantidade
	var pedido *domain.Pedido
	err = s.rest.WithTxVendas(ctx, s.vendas, func(rest *repository.RestauranteRepository, vendas *repository.VendaRepository) error {
		if nova <= eps {
			if err := vendas.RemoverItemConta(ctx, im.ID); err != nil {
				return err
			}
		} else {
			if err := vendas.AlterarQuantidadeItemConta(ctx, im.ID, nova); err != nil {
				return err
			}
			if im.Enviada > 0 {
				if err := rest.DefinirQuantidadeEnviada(ctx, im.ID, enviada-anularEnviada); err != nil {
					return err
				}
			}
		}
		if anularEnviada <= eps {
			return nil
		}
		aut := autorizador.ID
		criado, err := rest.CriarPedido(ctx, domain.Pedido{
			ContaID: im.ContaID, Tipo: domain.TipoPedidoAnulacao, Mesa: nomeConta(m), GarcomID: p.GarcomID,
			Motivo: &motivo, AutorizadoPor: &aut,
			Linhas: []domain.LinhaPedido{{ProdutoID: im.ProdutoID, Descricao: im.Produto, Unidade: im.Unidade,
				Quantidade: anularEnviada, ImpressoraID: r.destino(im.CategoriaID)}},
		})
		pedido = &criado
		return err
	})
	if err != nil {
		return ResultadoAnulacao{}, err
	}
	if pedido == nil {
		return ResultadoAnulacao{}, nil
	}
	res := s.imprimirPedido(ctx, *pedido, r, nil, false)
	return ResultadoAnulacao{Anulacao: &res}, nil
}

// --- Dividir ---------------------------------------------------------------------

type ParteDivisao struct {
	ItemID     int64
	Quantidade float64
}

// DividirConta passa as quantidades escolhidas para uma conta nova da mesma
// mesa ("5 (2)"), que se fecha à parte. O que já foi enviado continua
// enviado, nas duas contas. A divisão em partes iguais não precisa disto:
// faz-se ao imprimir a conta, com o número de pessoas.
func (s *RestauranteService) DividirConta(ctx context.Context, contaID int64, partes []ParteDivisao, etiqueta string) (domain.MesaAberta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	etiqueta, err := limparEtiqueta(etiqueta)
	if err != nil {
		return domain.MesaAberta{}, err
	}

	m, err := s.obterMesa(ctx, contaID)
	if err != nil {
		return domain.MesaAberta{}, err
	}
	itens, err := s.rest.ListarItens(ctx, contaID)
	if err != nil {
		return domain.MesaAberta{}, err
	}
	porID := map[int64]domain.ItemMesa{}
	var totalQtd float64
	for _, i := range itens {
		porID[i.ID] = i
		totalQtd += i.Quantidade
	}
	if len(partes) == 0 {
		return domain.MesaAberta{}, fmt.Errorf("escolha o que passa para a nova conta")
	}
	vistos := map[int64]bool{}
	var movida float64
	for _, p := range partes {
		i, ok := porID[p.ItemID]
		if !ok {
			return domain.MesaAberta{}, fmt.Errorf("o item %d não é desta mesa", p.ItemID)
		}
		if vistos[p.ItemID] {
			return domain.MesaAberta{}, fmt.Errorf("o item %q aparece duas vezes", i.Produto)
		}
		vistos[p.ItemID] = true
		if p.Quantidade <= eps || p.Quantidade > i.Quantidade+eps {
			return domain.MesaAberta{}, fmt.Errorf("quantidade de %q tem de estar entre 0 e %s", i.Produto, qtd(i.Quantidade))
		}
		if i.Desconto > 0 && p.Quantidade < i.Quantidade-eps {
			return domain.MesaAberta{}, fmt.Errorf("%q tem desconto — passe a linha inteira", i.Produto)
		}
		movida += p.Quantidade
	}
	if movida >= totalQtd-eps {
		return domain.MesaAberta{}, fmt.Errorf("a divisão tem de deixar alguma coisa na conta original")
	}

	abertas, err := s.rest.ListarMesasAbertas(ctx, domain.NegocioID)
	if err != nil {
		return domain.MesaAberta{}, err
	}
	nome := proximaConta(abertas, m.Mesa)

	var novaID int64
	err = s.rest.WithTxVendas(ctx, s.vendas, func(rest *repository.RestauranteRepository, vendas *repository.VendaRepository) error {
		orig, err := vendas.ObterConta(ctx, contaID)
		if err != nil {
			return err
		}
		c, err := vendas.CriarConta(ctx, domain.NegocioID, orig.ClienteID, rotuloMesa(nome), m.GarcomID)
		if err != nil {
			return err
		}
		novaID = c.ID
		if err := rest.DefinirContaMesa(ctx, c.ID, nome, m.GarcomID, 1); err != nil {
			return err
		}
		if err := rest.DefinirEtiquetaMesa(ctx, c.ID, etiqueta); err != nil {
			return err
		}
		for _, p := range partes {
			i := porID[p.ItemID]
			if p.Quantidade >= i.Quantidade-eps {
				if err := vendas.TransferirItemConta(ctx, i.ID, c.ID); err != nil {
					return err
				}
				continue
			}
			ic, err := vendas.ObterItemConta(ctx, i.ID)
			if err != nil {
				return err
			}
			if err := vendas.AlterarQuantidadeItemConta(ctx, i.ID, i.Quantidade-p.Quantidade); err != nil {
				return err
			}
			novo, err := vendas.CriarItemConta(ctx, domain.ItemConta{
				ContaID: c.ID, ProdutoID: i.ProdutoID, UnidadeProdutoID: i.UnidadeProdutoID, Quantidade: p.Quantidade,
				PrecoUnitarioAplicado: i.PrecoUnitario, UtilizadorID: ic.UtilizadorID,
			})
			if err != nil {
				return err
			}
			// O enviado vai primeiro com a parte que sai; o que fica por
			// enviar fica na conta original.
			env := math.Min(i.Enviada, p.Quantidade)
			if env > 0 {
				if err := rest.DefinirQuantidadeEnviada(ctx, novo.ID, env); err != nil {
					return err
				}
				if err := rest.DefinirQuantidadeEnviada(ctx, i.ID, i.Enviada-env); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return domain.MesaAberta{}, err
	}
	return s.obterMesa(ctx, novaID)
}

// --- Conta da mesa -----------------------------------------------------------------

type ContaImpressa struct {
	Total     float64
	Pessoas   int64
	PorPessoa float64
	// Texto é a conta a 32 colunas (impressora do aparelho, pré-visualização).
	Texto string
	// Impressao: resultado na impressora de rede, se foi escolhida uma.
	Impressao *ResultadoImpressao
}

// ImprimirConta imprime a conta da mesa para o cliente conferir, com o
// valor por pessoa (900 para 3 pessoas = 300 cada). pessoas 0 mantém o que
// a mesa tem; impressoraID nil devolve só o texto para o aparelho.
func (s *RestauranteService) ImprimirConta(ctx context.Context, contaID, pessoas int64, impressoraID *int64) (ContaImpressa, error) {
	m, err := s.obterMesa(ctx, contaID)
	if err != nil {
		return ContaImpressa{}, err
	}
	if pessoas > 0 && pessoas != m.Pessoas {
		if err := s.rest.DefinirContaMesa(ctx, contaID, m.Mesa, m.GarcomID, pessoas); err != nil {
			return ContaImpressa{}, err
		}
		m.Pessoas = pessoas
	}
	itens, err := s.rest.ListarItens(ctx, contaID)
	if err != nil {
		return ContaImpressa{}, err
	}
	if len(itens) == 0 {
		return ContaImpressa{}, fmt.Errorf("a mesa ainda não tem nada lançado")
	}
	negocio, err := s.negocio.Obter(ctx)
	if err != nil {
		return ContaImpressa{}, err
	}
	cfg, err := s.configs.Listar(ctx)
	if err != nil {
		return ContaImpressa{}, err
	}
	loja := export.DadosRecibo{
		NegocioNome: negocio.Nome, LojaNUIT: cfg["loja_nuit"], LojaEndereco: cfg["loja_endereco"],
		LojaTelefone: cfg["loja_telefone"], LojaEmail: cfg["loja_email"],
	}
	var total float64
	for _, i := range itens {
		total += i.Subtotal
	}
	res := ContaImpressa{Total: total, Pessoas: m.Pessoas, PorPessoa: total / float64(m.Pessoas)}
	agora := time.Now()
	res.Texto = talaoConta(loja, m, itens, res, agora, 32).TextoSimples()
	if impressoraID != nil {
		imp, err := s.rest.ObterImpressora(ctx, *impressoraID)
		if err != nil {
			return ContaImpressa{}, err
		}
		ri := ResultadoImpressao{ImpressoraID: imp.ID, Nome: imp.Nome}
		if err := s.impressor.Enviar(ctx, imp.Endereco, int(imp.Porta), talaoConta(loja, m, itens, res, agora, int(imp.Colunas)).ESCPOS()); err != nil {
			ri.Erro = err.Error()
		}
		res.Impressao = &ri
	}
	return res, nil
}

func talaoConta(loja export.DadosRecibo, m domain.MesaAberta, itens []domain.ItemMesa, c ContaImpressa, agora time.Time, colunas int) *impressao.Talao {
	t := impressao.NovoTalao(colunas)
	t.Titulo(loja.NegocioNome)
	for _, l := range loja.LinhasLoja() {
		t.Centro(l)
	}
	t.Separador()
	t.Titulo("CONTA · " + strings.ToUpper(rotuloMesa(m.Mesa)))
	if m.Etiqueta != "" {
		t.Centro(m.Etiqueta)
	}
	t.Centro("Garçom " + m.GarcomNome + " · " + agora.Format("02/01/2006 15:04"))
	t.Separador()
	for _, i := range itens {
		t.Par(descricaoLinha(domain.LinhaPedido{Descricao: i.Produto, Unidade: i.Unidade, Quantidade: i.Quantidade}),
			export.FormatarValor(i.Quantidade*i.PrecoUnitario), false)
		if i.Quantidade != 1 {
			t.Texto("   " + export.FormatarValor(i.PrecoUnitario) + " cada")
		}
		if i.Desconto > 0 {
			t.Par("   desconto", "-"+export.FormatarValor(i.Desconto), false)
		}
	}
	t.Separador()
	t.Par("TOTAL (MT)", export.FormatarValor(c.Total), true)
	if c.Pessoas > 1 {
		t.Par(fmt.Sprintf("%d pessoas · cada", c.Pessoas), export.FormatarValor(c.PorPessoa), true)
	}
	t.Separador()
	t.Centro("Consulta — não serve de recibo")
	return t
}

// FecharMesa cobra a mesa no caixa. Só com o caixa aberto, para o
// dinheiro entrar na sessão de caixa certa.
func (s *RestauranteService) FecharMesa(ctx context.Context, contaID int64, p FecharContaParams) (VendaCompleta, AlertaLimiteCredito, error) {
	if _, ok, err := s.rest.SessaoAberta(ctx); err != nil {
		return VendaCompleta{}, AlertaLimiteCredito{}, err
	} else if !ok {
		return VendaCompleta{}, AlertaLimiteCredito{}, ErrCaixaFechado
	}
	if _, err := s.obterMesa(ctx, contaID); err != nil {
		return VendaCompleta{}, AlertaLimiteCredito{}, err
	}
	return s.contas.FecharConta(ctx, contaID, p)
}

// --- Impressoras -------------------------------------------------------------------

func (s *RestauranteService) ListarImpressoras(ctx context.Context) ([]domain.Impressora, error) {
	return s.rest.ListarImpressoras(ctx)
}

// GuardarImpressora cria (ID 0) ou altera uma impressora de rede.
func (s *RestauranteService) GuardarImpressora(ctx context.Context, i domain.Impressora) (domain.Impressora, error) {
	i.Nome = strings.TrimSpace(i.Nome)
	i.Endereco = strings.TrimSpace(i.Endereco)
	if i.Nome == "" {
		return domain.Impressora{}, fmt.Errorf("dê um nome à impressora (ex: Cozinha)")
	}
	if i.Endereco == "" {
		return domain.Impressora{}, fmt.Errorf("diga o endereço IP da impressora (ex: 192.168.1.100)")
	}
	if i.Porta == 0 {
		i.Porta = 9100
	}
	if i.Porta < 1 || i.Porta > 65535 {
		return domain.Impressora{}, fmt.Errorf("porta inválida")
	}
	if i.Colunas == 0 {
		i.Colunas = 48
	}
	if i.Colunas < 24 || i.Colunas > 64 {
		return domain.Impressora{}, fmt.Errorf("colunas inválidas (48 no papel de 80 mm, 32 no de 58 mm)")
	}
	if i.Funcao != domain.ImpressoraBalcao && i.Funcao != domain.ImpressoraPreparacao {
		return domain.Impressora{}, fmt.Errorf("escolha se a impressora é do balcão ou de preparação")
	}
	if i.ID == 0 {
		return s.rest.CriarImpressora(ctx, i)
	}
	if err := s.rest.AtualizarImpressora(ctx, i); err != nil {
		return domain.Impressora{}, err
	}
	return s.rest.ObterImpressora(ctx, i.ID)
}

// TestarImpressora imprime um talão de teste.
func (s *RestauranteService) TestarImpressora(ctx context.Context, id int64) error {
	imp, err := s.rest.ObterImpressora(ctx, id)
	if err != nil {
		return err
	}
	t := impressao.NovoTalao(int(imp.Colunas)).Titulo("TESTE").Centro(imp.Nome).
		Centro(time.Now().Format("02/01/2006 15:04")).Separador().
		Texto("Acentos: ã õ ç é à ê ó ú — Mesa nº 5").Separador()
	return s.impressor.Enviar(ctx, imp.Endereco, int(imp.Porta), t.ESCPOS())
}

// RotasImpressao: categoria → impressora de preparação.
func (s *RestauranteService) RotasImpressao(ctx context.Context) (map[int64]int64, error) {
	return s.rest.ImpressoraPorCategoria(ctx)
}

// DefinirRotaCategoria diz para que impressora de preparação vão os produtos
// de uma categoria (nil = só o balcão).
func (s *RestauranteService) DefinirRotaCategoria(ctx context.Context, categoriaID int64, impressoraID *int64) error {
	if impressoraID != nil {
		imp, err := s.rest.ObterImpressora(ctx, *impressoraID)
		if err != nil {
			return err
		}
		if imp.Funcao != domain.ImpressoraPreparacao {
			return fmt.Errorf("%q é do balcão — o balcão já recebe todos os pedidos", imp.Nome)
		}
	}
	return s.rest.DefinirImpressoraDaCategoria(ctx, categoriaID, impressoraID)
}
