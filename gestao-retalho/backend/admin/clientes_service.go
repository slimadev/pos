package admin

import (
	"context"
	"time"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// ClientesService serve o ecrã "Clientes e dívidas" do cmd/admin. O saldo
// em dívida de cada cliente é somado aqui, numa só ligação à loja, para a
// lista não precisar de uma chamada por cliente.
type ClientesService struct {
	lojas *repository.LojaRepository
}

func NewClientesService(lojas *repository.LojaRepository) *ClientesService {
	return &ClientesService{lojas: lojas}
}

type ClienteResumo struct {
	Cliente        domain.Cliente
	TotalEmAberto  float64
	DividasAbertas int
	// DiasDividaMaisAntiga é 0 quando o cliente não deve nada.
	DiasDividaMaisAntiga int
	LimiteExcedido       bool
}

type FichaCliente struct {
	Cliente domain.Cliente
	Dividas []domain.Divida
	Vendas  []domain.Venda
}

// Listar devolve os clientes identificados (o anónimo fica de fora), com o
// que cada um deve.
func (s *ClientesService) Listar(ctx context.Context) ([]ClienteResumo, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) ([]ClienteResumo, error) {
		clientes, err := a.ListarClientes()
		if err != nil {
			return nil, err
		}
		abertas, err := a.ListarDividasAbertas()
		if err != nil {
			return nil, err
		}

		porCliente := make(map[int64]*ClienteResumo)
		out := make([]ClienteResumo, 0, len(clientes))
		for _, c := range clientes {
			if c.IsAnonimo || c.ID == domain.ClienteAnonimoID {
				continue
			}
			out = append(out, ClienteResumo{Cliente: c})
		}
		for i := range out {
			porCliente[out[i].Cliente.ID] = &out[i]
		}

		agora := time.Now()
		for _, d := range abertas {
			r, ok := porCliente[d.ClienteID]
			if !ok {
				continue
			}
			r.TotalEmAberto += d.ValorEmAberto
			r.DividasAbertas++
			if dias := int(agora.Sub(d.CriadoEm).Hours() / 24); dias > r.DiasDividaMaisAntiga {
				r.DiasDividaMaisAntiga = dias
			}
		}
		for i := range out {
			if lim := out[i].Cliente.LimiteCredito; lim != nil && out[i].TotalEmAberto > *lim {
				out[i].LimiteExcedido = true
			}
		}
		return out, nil
	})
}

func (s *ClientesService) ObterFicha(ctx context.Context, clienteID int64) (FichaCliente, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (FichaCliente, error) {
		cliente, err := a.ObterCliente(clienteID)
		if err != nil {
			return FichaCliente{}, err
		}
		dividas, err := a.ListarDividasPorCliente(clienteID)
		if err != nil {
			return FichaCliente{}, err
		}
		vendas, err := a.ListarVendasPorCliente(clienteID)
		if err != nil {
			return FichaCliente{}, err
		}
		return FichaCliente{Cliente: cliente, Dividas: dividas, Vendas: vendas}, nil
	})
}

func (s *ClientesService) Criar(ctx context.Context, c domain.Cliente) (domain.Cliente, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (domain.Cliente, error) { return a.CriarCliente(c) })
}

func (s *ClientesService) Atualizar(ctx context.Context, c domain.Cliente) error {
	_, err := comLoja(ctx, s.lojas, func(a *api.App) (struct{}, error) { return struct{}{}, a.AtualizarCliente(c) })
	return err
}

func (s *ClientesService) Ativar(ctx context.Context, id int64) error {
	_, err := comLoja(ctx, s.lojas, func(a *api.App) (struct{}, error) { return struct{}{}, a.AtivarCliente(id) })
	return err
}

func (s *ClientesService) Desativar(ctx context.Context, id int64) error {
	_, err := comLoja(ctx, s.lojas, func(a *api.App) (struct{}, error) { return struct{}{}, a.DesativarCliente(id) })
	return err
}

func (s *ClientesService) ListarPagamentosDivida(ctx context.Context, dividaID int64) ([]domain.PagamentoDivida, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) ([]domain.PagamentoDivida, error) { return a.ListarPagamentosDivida(dividaID) })
}

func (s *ClientesService) RegistarPagamentoDivida(ctx context.Context, p usecase.RegistarPagamentoDividaParams) (api.RegistarPagamentoDividaResponse, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (api.RegistarPagamentoDividaResponse, error) {
		return a.RegistarPagamentoDivida(p)
	})
}
