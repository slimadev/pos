package admin

import (
	"context"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// RestauranteService serve o ecrã "Restaurante" do admin: ligar o modo
// restaurante, as impressoras de rede, para onde vai cada categoria, os
// gerentes e o histórico dos fechos de caixa.
type RestauranteService struct {
	lojas *repository.LojaRepository
}

func NewRestauranteService(lojas *repository.LojaRepository) *RestauranteService {
	return &RestauranteService{lojas: lojas}
}

type ConfigRestaurante struct {
	Ativo       bool
	Impressoras []domain.Impressora
	Rotas       []api.RotaCategoria
	Servidor    usecase.ServidorAparelhos
}

func (s *RestauranteService) Obter(ctx context.Context) (ConfigRestaurante, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (ConfigRestaurante, error) {
		est, err := a.ObterEstadoConfig()
		if err != nil {
			return ConfigRestaurante{}, err
		}
		c := ConfigRestaurante{}
		for _, m := range est.ModulosAtivos {
			c.Ativo = c.Ativo || m == domain.ModuloMesas
		}
		if c.Impressoras, err = a.ListarImpressoras(); err != nil {
			return c, err
		}
		if c.Rotas, err = a.ListarRotasImpressao(); err != nil {
			return c, err
		}
		c.Servidor, err = a.ConfigService.ObterServidorAparelhos(ctx)
		return c, err
	})
}

func (s *RestauranteService) DefinirAtivo(ctx context.Context, ativo bool) error {
	_, err := comLoja(ctx, s.lojas, func(a *api.App) (struct{}, error) {
		if err := a.DefinirModoRestaurante(ativo); err != nil {
			return struct{}{}, err
		}
		// O PDV do balcão passa a atender os aparelhos (a partir do
		// próximo arranque do PDV).
		cfg, err := a.ConfigService.ObterServidorAparelhos(ctx)
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, a.ConfigService.DefinirServidorAparelhos(ctx, ativo, cfg.Porta)
	})
	return err
}

func (s *RestauranteService) GuardarImpressora(ctx context.Context, i domain.Impressora) (domain.Impressora, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (domain.Impressora, error) { return a.GuardarImpressora(i) })
}

func (s *RestauranteService) TestarImpressora(ctx context.Context, id int64) error {
	_, err := comLoja(ctx, s.lojas, func(a *api.App) (struct{}, error) { return struct{}{}, a.TestarImpressora(id) })
	return err
}

func (s *RestauranteService) DefinirRota(ctx context.Context, categoriaID int64, impressoraID *int64) error {
	_, err := comLoja(ctx, s.lojas, func(a *api.App) (struct{}, error) {
		return struct{}{}, a.DefinirRotaCategoria(categoriaID, impressoraID)
	})
	return err
}

func (s *RestauranteService) DefinirGerente(ctx context.Context, id int64, gerente bool) error {
	_, err := comLoja(ctx, s.lojas, func(a *api.App) (struct{}, error) { return struct{}{}, a.DefinirGerente(id, gerente) })
	return err
}

func (s *RestauranteService) DefinirCaixa(ctx context.Context, id int64, caixa bool) error {
	_, err := comLoja(ctx, s.lojas, func(a *api.App) (struct{}, error) { return struct{}{}, a.DefinirCaixa(id, caixa) })
	return err
}

func (s *RestauranteService) HistoricoCaixa(ctx context.Context) ([]domain.SessaoCaixa, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) ([]domain.SessaoCaixa, error) { return a.HistoricoCaixa(60) })
}

func (s *RestauranteService) ObterFechoCaixa(ctx context.Context, id int64) (usecase.FechoSessao, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (usecase.FechoSessao, error) { return a.ObterFechoCaixa(id) })
}

func (s *RestauranteService) NovaChaveServidor(ctx context.Context) (usecase.ServidorAparelhos, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (usecase.ServidorAparelhos, error) {
		if _, err := a.ConfigService.NovaChaveServidor(ctx); err != nil {
			return usecase.ServidorAparelhos{}, err
		}
		return a.ConfigService.ObterServidorAparelhos(ctx)
	})
}
