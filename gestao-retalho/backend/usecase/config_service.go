package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	appconfig "gestao-retalho/backend/config"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// ConfigService cobre o wizard de primeiro arranque e a leitura de
// configuração em runtime (SDD secção 9). "Configurado" é decidido pela
// chave `wizard_concluido` em `configuracoes` — a base é sempre a fonte da
// verdade em runtime (9.3: "consultável por qualquer service"); config.json
// é escrito como espelho, não relido depois.
type ConfigService struct {
	negocios     *repository.NegocioRepository
	configs      *repository.ConfigRepository
	utilizadores *repository.UtilizadorRepository
	categorias   *repository.CategoriaRepository
	armazens     *ArmazemService
	produtos     *ProdutoService
}

func NewConfigService(
	negocios *repository.NegocioRepository,
	configs *repository.ConfigRepository,
	utilizadores *repository.UtilizadorRepository,
	categorias *repository.CategoriaRepository,
	armazens *ArmazemService,
	produtos *ProdutoService,
) *ConfigService {
	return &ConfigService{
		negocios:     negocios,
		configs:      configs,
		utilizadores: utilizadores,
		categorias:   categorias,
		armazens:     armazens,
		produtos:     produtos,
	}
}

const chaveWizardConcluido = "wizard_concluido"

func (s *ConfigService) EstaConfigurado(ctx context.Context) (bool, error) {
	valor, ok, err := s.configs.Obter(ctx, chaveWizardConcluido)
	if err != nil {
		return false, err
	}
	return ok && valor == "true", nil
}

// EstadoConfig é o que o frontend precisa em cada arranque (depois do
// wizard) para montar o menu dinâmico (SDD 9.5) e aplicar o tema (9.6).
type EstadoConfig struct {
	Negocio       domain.Negocio
	ModulosAtivos []domain.Modulo
	Tema          domain.Tema
	Logo          string // "" = sem logótipo (ver FinalizarWizardParams.Logo)
	PapelRecibo   string // "80", "58" ou "a5" (ver ReciboService)
}

func (s *ConfigService) ObterEstado(ctx context.Context) (EstadoConfig, error) {
	negocio, err := s.negocios.Obter(ctx)
	if err != nil {
		return EstadoConfig{}, err
	}
	todas, err := s.configs.Listar(ctx)
	if err != nil {
		return EstadoConfig{}, err
	}

	// Nunca nil mesmo com zero módulos ativos (ex: "Personalizado" sem
	// nenhum escolhido) — um slice nil marshalla para `null` em JSON, e o
	// frontend faz `config.ModulosAtivos.includes(...)` sem guarda nenhuma.
	modulos := make([]domain.Modulo, 0)
	for _, m := range domain.TodosModulos {
		if todas["modulo_"+string(m)] == "true" {
			modulos = append(modulos, m)
		}
	}

	tema := domain.Tema(todas["tema"])
	if tema == "" {
		tema = domain.TemaAdmin
	}

	papel := todas["recibo_papel"]
	if papel == "" {
		papel = PapelRecibo80
	}
	return EstadoConfig{Negocio: negocio, ModulosAtivos: modulos, Tema: tema, Logo: todas["logo"], PapelRecibo: papel}, nil
}

// FinalizarWizardParams junta os quatro passos do wizard (SDD 9.2) numa só
// chamada — não há estado a meio caminho gravado entre passos, o frontend
// mantém o progresso e só chama isto no fim.
type FinalizarWizardParams struct {
	NegocioNome          string
	TipoNegocio          domain.TipoNegocio
	ModulosAtivos        []domain.Modulo // só usado quando TipoNegocio == Personalizado
	ArmazemPrincipalNome string
	Logo                 string // data URL opcional (ex: "data:image/png;base64,..."), "" = sem logótipo
	AdminNome            string
	AdminUsername        string
	AdminPassword        string
}

func (s *ConfigService) FinalizarWizard(ctx context.Context, p FinalizarWizardParams) error {
	if p.NegocioNome == "" {
		return fmt.Errorf("nome do negócio é obrigatório")
	}
	if p.ArmazemPrincipalNome == "" {
		return fmt.Errorf("nome do armazém principal é obrigatório")
	}
	if p.AdminNome == "" || p.AdminUsername == "" {
		return fmt.Errorf("nome e username do administrador são obrigatórios")
	}
	if len(p.AdminPassword) < 6 {
		return fmt.Errorf("password do administrador deve ter pelo menos 6 caracteres")
	}

	modulos, tema, err := resolverModulosETema(p.TipoNegocio, p.ModulosAtivos)
	if err != nil {
		return err
	}

	if _, err := s.negocios.Atualizar(ctx, p.NegocioNome, p.TipoNegocio); err != nil {
		return fmt.Errorf("gravar negócio: %w", err)
	}

	ativos := make(map[domain.Modulo]bool, len(modulos))
	for _, m := range modulos {
		ativos[m] = true
	}
	for _, m := range domain.TodosModulos {
		valor := "false"
		if ativos[m] {
			valor = "true"
		}
		if err := s.configs.Definir(ctx, "modulo_"+string(m), valor); err != nil {
			return fmt.Errorf("gravar módulo %s: %w", m, err)
		}
	}
	if err := s.configs.Definir(ctx, "tema", string(tema)); err != nil {
		return fmt.Errorf("gravar tema: %w", err)
	}
	if err := s.configs.Definir(ctx, "logo", p.Logo); err != nil {
		return fmt.Errorf("gravar logótipo: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(p.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("gerar hash da password: %w", err)
	}
	if err := s.utilizadores.AtualizarAdmin(ctx, domain.UtilizadorAdminSeedID, p.AdminNome, p.AdminUsername, string(hash)); err != nil {
		return fmt.Errorf("gravar administrador: %w", err)
	}

	if _, err := s.armazens.Criar(ctx, domain.Armazem{Nome: p.ArmazemPrincipalNome, Tipo: domain.TipoArmazemPrincipal}); err != nil {
		return fmt.Errorf("criar armazém principal: %w", err)
	}

	if err := aplicarSeed(ctx, p.TipoNegocio, s.categorias, s.produtos); err != nil {
		return fmt.Errorf("aplicar seed: %w", err)
	}

	// wizard_concluido só é gravado no fim — se algo acima falhar, o
	// utilizador reabre o wizard e tenta de novo em vez de ficar com a app
	// "meio configurada" mas marcada como pronta.
	if err := s.configs.Definir(ctx, chaveWizardConcluido, "true"); err != nil {
		return fmt.Errorf("concluir wizard: %w", err)
	}

	modulosStr := make([]string, len(modulos))
	for i, m := range modulos {
		modulosStr[i] = string(m)
	}
	_ = appconfig.SaveAppConfig(appconfig.AppConfig{
		NegocioNome:   p.NegocioNome,
		TipoNegocio:   string(p.TipoNegocio),
		ModulosAtivos: modulosStr,
		Tema:          string(tema),
		InstaladoEm:   time.Now().UTC(),
	}) // espelho best-effort (SDD 9.3) — a base já é a fonte da verdade acima.

	return nil
}

func resolverModulosETema(tipo domain.TipoNegocio, modulosPersonalizados []domain.Modulo) ([]domain.Modulo, domain.Tema, error) {
	if tipo == domain.TipoNegocioPersonalizado {
		return modulosPersonalizados, domain.TemaAdmin, nil
	}
	for _, preset := range domain.PresetsNegocio {
		if preset.Tipo == tipo {
			return preset.Modulos, preset.Tema, nil
		}
	}
	return nil, "", fmt.Errorf("tipo de negócio desconhecido: %q", tipo)
}

// tamanhoMaximoLogo limita o data URL do logótipo (~1,5 MB de imagem em
// base64) — vive numa linha de configuracoes e é lido a cada arranque.
const tamanhoMaximoLogo = 2_200_000

// AtualizarIdentidade muda o nome do negócio, o logótipo e o papel da
// impressora de recibos depois do assistente inicial; o tipo de negócio (e
// com ele módulos e tema) não muda aqui. logo "" remove o logótipo.
func (s *ConfigService) AtualizarIdentidade(ctx context.Context, nome, logo, papelRecibo string) (domain.Negocio, error) {
	switch papelRecibo {
	case "":
		papelRecibo = PapelRecibo80
	case PapelRecibo80, PapelRecibo58, PapelReciboA5:
	default:
		return domain.Negocio{}, fmt.Errorf("papel do recibo inválido: %q", papelRecibo)
	}
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return domain.Negocio{}, fmt.Errorf("nome do negócio é obrigatório")
	}
	if logo != "" && !strings.HasPrefix(logo, "data:image/") {
		return domain.Negocio{}, fmt.Errorf("o logótipo tem de ser uma imagem")
	}
	if len(logo) > tamanhoMaximoLogo {
		return domain.Negocio{}, fmt.Errorf("logótipo demasiado grande — escolha uma imagem até 1,5 MB")
	}
	atual, err := s.negocios.Obter(ctx)
	if err != nil {
		return domain.Negocio{}, err
	}
	negocio, err := s.negocios.Atualizar(ctx, nome, atual.TipoNegocio)
	if err != nil {
		return domain.Negocio{}, fmt.Errorf("gravar negócio: %w", err)
	}
	if err := s.configs.Definir(ctx, "logo", logo); err != nil {
		return domain.Negocio{}, fmt.Errorf("gravar logótipo: %w", err)
	}
	if err := s.configs.Definir(ctx, "recibo_papel", papelRecibo); err != nil {
		return domain.Negocio{}, fmt.Errorf("gravar papel do recibo: %w", err)
	}
	return negocio, nil
}
