package usecase

import (
	"context"
	"fmt"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// DividaService cobre a vida de uma dívida depois de criada (SDD 4.11) —
// a criação em si acontece atomicamente ao fechar uma conta (Fase 4,
// ContaService.FecharConta); pagar é sempre um fluxo à parte.
type DividaService struct {
	dividas *repository.DividaRepository
	cliente *repository.ClienteRepository
}

func NewDividaService(dividas *repository.DividaRepository, cliente *repository.ClienteRepository) *DividaService {
	return &DividaService{dividas: dividas, cliente: cliente}
}

func (s *DividaService) ListarPorCliente(ctx context.Context, clienteID int64) ([]domain.Divida, error) {
	return s.dividas.ListarPorCliente(ctx, clienteID)
}

func (s *DividaService) ListarAbertas(ctx context.Context, negocioID int64) ([]domain.Divida, error) {
	return s.dividas.ListarAbertas(ctx, negocioID)
}

// ListarAntigas cobre o "alerta de dívidas antigas (> 30 dias)" do SDD
// 4.11 — o mesmo painel que o dashboard admin (tela `1c`) vai reutilizar.
func (s *DividaService) ListarAntigas(ctx context.Context, negocioID int64, diasMinimos int) ([]domain.Divida, error) {
	abertas, err := s.dividas.ListarAbertas(ctx, negocioID)
	if err != nil {
		return nil, err
	}
	limite := time.Now().AddDate(0, 0, -diasMinimos)
	antigas := make([]domain.Divida, 0, len(abertas))
	for _, d := range abertas {
		if d.CriadoEm.Before(limite) {
			antigas = append(antigas, d)
		}
	}
	return antigas, nil
}

func (s *DividaService) ListarPagamentos(ctx context.Context, dividaID int64) ([]domain.PagamentoDivida, error) {
	return s.dividas.ListarPagamentos(ctx, dividaID)
}

type RegistarPagamentoDividaParams struct {
	DividaID     int64
	Valor        float64
	Metodo       domain.MetodoPagamento
	UtilizadorID int64
}

// toleranciaDivida usa a mesma folga de arredondamento que toleranciaPagamento
// (venda_service.go) — evita que 0.1+0.2 != 0.3 deixe uma dívida "quase
// paga" presa em parcial para sempre.
const toleranciaDivida = 0.01

// RegistarPagamento é o fluxo independente de nova venda do SDD 4.11 —
// atualiza valor_pago/valor_em_aberto/status da dívida e grava o
// pagamento, atomicamente.
func (s *DividaService) RegistarPagamento(ctx context.Context, p RegistarPagamentoDividaParams) (domain.Divida, domain.PagamentoDivida, error) {
	if p.Valor <= 0 {
		return domain.Divida{}, domain.PagamentoDivida{}, fmt.Errorf("valor do pagamento deve ser positivo")
	}

	divida, err := s.dividas.Obter(ctx, p.DividaID)
	if err != nil {
		return domain.Divida{}, domain.PagamentoDivida{}, err
	}
	if divida.Status == domain.StatusDividaPaga {
		return domain.Divida{}, domain.PagamentoDivida{}, fmt.Errorf("dívida já está paga")
	}
	if p.Valor > divida.ValorEmAberto+toleranciaDivida {
		return domain.Divida{}, domain.PagamentoDivida{}, fmt.Errorf("pagamento (%.2f) excede o saldo em aberto (%.2f)", p.Valor, divida.ValorEmAberto)
	}

	novoValorPago := divida.ValorPago + p.Valor
	novoValorEmAberto := divida.ValorOriginal - novoValorPago
	status := domain.StatusDividaParcial
	if novoValorEmAberto <= toleranciaDivida {
		novoValorEmAberto = 0
		status = domain.StatusDividaPaga
	}

	var dividaAtualizada domain.Divida
	var pagamento domain.PagamentoDivida
	err = s.dividas.WithTx(ctx, func(tx *repository.DividaRepository) error {
		var err error
		dividaAtualizada, err = tx.Atualizar(ctx, p.DividaID, novoValorPago, novoValorEmAberto, status)
		if err != nil {
			return fmt.Errorf("atualizar dívida: %w", err)
		}
		pagamento, err = tx.CriarPagamento(ctx, domain.PagamentoDivida{
			DividaID:     p.DividaID,
			Valor:        p.Valor,
			Metodo:       p.Metodo,
			UtilizadorID: p.UtilizadorID,
		})
		if err != nil {
			return fmt.Errorf("gravar pagamento: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Divida{}, domain.PagamentoDivida{}, err
	}
	return dividaAtualizada, pagamento, nil
}

// AlertaLimiteCredito é o "bloqueio opcional (alerta, não obrigatório)" do
// SDD 4.9 — nunca impede a dívida de ser criada, só informa que o cliente
// já passou do limite combinado, para o operador decidir.
type AlertaLimiteCredito struct {
	Excedido      bool
	LimiteCredito float64
	TotalEmAberto float64
}

// VerificarLimiteCredito soma as dívidas já abertas do cliente com um
// valor adicional (o que uma conta prestes a fechar deixaria em aberto) e
// compara com o limite_credito do cliente, se este tiver algum definido.
func (s *DividaService) VerificarLimiteCredito(ctx context.Context, clienteID int64, valorAdicional float64) (AlertaLimiteCredito, error) {
	cliente, err := s.cliente.Obter(ctx, clienteID)
	if err != nil {
		return AlertaLimiteCredito{}, err
	}
	if cliente.LimiteCredito == nil {
		return AlertaLimiteCredito{}, nil
	}

	existentes, err := s.dividas.ListarPorCliente(ctx, clienteID)
	if err != nil {
		return AlertaLimiteCredito{}, err
	}
	var totalEmAberto float64
	for _, d := range existentes {
		if d.Status != domain.StatusDividaPaga {
			totalEmAberto += d.ValorEmAberto
		}
	}
	totalEmAberto += valorAdicional

	return AlertaLimiteCredito{
		Excedido:      totalEmAberto > *cliente.LimiteCredito,
		LimiteCredito: *cliente.LimiteCredito,
		TotalEmAberto: totalEmAberto,
	}, nil
}
