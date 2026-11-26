package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
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
	NUIT          string
	Endereco      string
	Telefone      string
	Email         string
	TaxaIVA       float64 // em %, 0 = isento
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
	return EstadoConfig{
		Negocio: negocio, ModulosAtivos: modulos, Tema: tema, Logo: todas["logo"], PapelRecibo: papel,
		NUIT: todas["loja_nuit"], Endereco: todas["loja_endereco"], Telefone: todas["loja_telefone"],
		Email: todas["loja_email"], TaxaIVA: taxaIVA(todas["iva_taxa"]),
	}, nil
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

// DadosLoja é o que se muda nas configurações da loja depois do
// assistente inicial. O tipo de negócio (e com ele módulos e tema) não
// muda aqui.
type DadosLoja struct {
	Nome        string
	Logo        string // data URL, "" remove o logótipo
	PapelRecibo string // "80", "58" ou "a5"
	NUIT        string // 9 algarismos, "" = por preencher
	Endereco    string
	Telefone    string
	Email       string
	TaxaIVA     float64 // em %, 0 = isento
}

// TaxaIVAPorOmissao é a taxa normal de IVA em Moçambique, usada enquanto a
// loja não define a sua.
const TaxaIVAPorOmissao = 16.0

// AtualizarIdentidade grava os dados da loja (nome, logótipo, papel do
// recibo, NUIT, contactos e taxa de IVA).
func (s *ConfigService) AtualizarIdentidade(ctx context.Context, d DadosLoja) (domain.Negocio, error) {
	switch d.PapelRecibo {
	case "":
		d.PapelRecibo = PapelRecibo80
	case PapelRecibo80, PapelRecibo58, PapelReciboA5:
	default:
		return domain.Negocio{}, fmt.Errorf("papel do recibo inválido: %q", d.PapelRecibo)
	}
	nome := strings.TrimSpace(d.Nome)
	if nome == "" {
		return domain.Negocio{}, fmt.Errorf("nome do negócio é obrigatório")
	}
	if d.Logo != "" && !strings.HasPrefix(d.Logo, "data:image/") {
		return domain.Negocio{}, fmt.Errorf("o logótipo tem de ser uma imagem")
	}
	if len(d.Logo) > tamanhoMaximoLogo {
		return domain.Negocio{}, fmt.Errorf("logótipo demasiado grande — escolha uma imagem até 1,5 MB")
	}
	nuit := strings.ReplaceAll(strings.TrimSpace(d.NUIT), " ", "")
	if nuit != "" && !nuitValido(nuit) {
		return domain.Negocio{}, fmt.Errorf("o NUIT tem de ter 9 algarismos")
	}
	email := strings.TrimSpace(d.Email)
	if email != "" && !emailValido(email) {
		return domain.Negocio{}, fmt.Errorf("email inválido: %q", email)
	}
	if d.TaxaIVA < 0 || d.TaxaIVA > 100 {
		return domain.Negocio{}, fmt.Errorf("a taxa de IVA tem de estar entre 0 e 100%%")
	}

	atual, err := s.negocios.Obter(ctx)
	if err != nil {
		return domain.Negocio{}, err
	}
	negocio, err := s.negocios.Atualizar(ctx, nome, atual.TipoNegocio)
	if err != nil {
		return domain.Negocio{}, fmt.Errorf("gravar negócio: %w", err)
	}
	valores := map[string]string{
		"logo":          d.Logo,
		"recibo_papel":  d.PapelRecibo,
		"loja_nuit":     nuit,
		"loja_endereco": strings.TrimSpace(d.Endereco),
		"loja_telefone": strings.TrimSpace(d.Telefone),
		"loja_email":    email,
		"iva_taxa":      strconv.FormatFloat(d.TaxaIVA, 'f', -1, 64),
	}
	for chave, valor := range valores {
		if err := s.configs.Definir(ctx, chave, valor); err != nil {
			return domain.Negocio{}, fmt.Errorf("gravar %s: %w", chave, err)
		}
	}
	return negocio, nil
}

func nuitValido(nuit string) bool {
	if len(nuit) != 9 {
		return false
	}
	for _, c := range nuit {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// emailValido só apanha enganos óbvios (falta de @ ou de domínio) — não
// tenta validar o endereço por completo.
func emailValido(email string) bool {
	arroba := strings.LastIndex(email, "@")
	return arroba > 0 && !strings.ContainsAny(email, " ,;") &&
		strings.Contains(email[arroba+1:], ".") && !strings.HasSuffix(email, ".")
}

// taxaIVA lê a taxa gravada; sem nada gravado, a taxa normal.
func taxaIVA(valor string) float64 {
	if valor == "" {
		return TaxaIVAPorOmissao
	}
	t, err := strconv.ParseFloat(valor, 64)
	if err != nil {
		return TaxaIVAPorOmissao
	}
	return t
}

// --- Servidor para os aparelhos dos garçons ------------------------------------

// PortaServidorPadrao é a porta em que o PC do balcão atende os aparelhos.
const PortaServidorPadrao = 8765

// ServidorAparelhos: se o PDV deste PC atende os aparelhos dos garçons
// (Sunmi…) na rede local, em que porta, e a chave que cada aparelho tem de
// apresentar — escreve-se uma vez em cada aparelho.
type ServidorAparelhos struct {
	Ativo bool
	Porta int
	Chave string
}

func (s *ConfigService) ObterServidorAparelhos(ctx context.Context) (ServidorAparelhos, error) {
	todas, err := s.configs.Listar(ctx)
	if err != nil {
		return ServidorAparelhos{}, err
	}
	sa := ServidorAparelhos{Ativo: todas["servidor_ativo"] == "1", Porta: PortaServidorPadrao, Chave: todas["servidor_chave"]}
	if p, err := strconv.Atoi(todas["servidor_porta"]); err == nil && p > 0 && p < 65536 {
		sa.Porta = p
	}
	if sa.Chave == "" {
		if sa.Chave, err = s.NovaChaveServidor(ctx); err != nil {
			return ServidorAparelhos{}, err
		}
	}
	return sa, nil
}

func (s *ConfigService) DefinirServidorAparelhos(ctx context.Context, ativo bool, porta int) error {
	if porta == 0 {
		porta = PortaServidorPadrao
	}
	if porta < 1024 || porta > 65535 {
		return fmt.Errorf("porta inválida (use entre 1024 e 65535)")
	}
	v := "0"
	if ativo {
		v = "1"
	}
	if err := s.configs.Definir(ctx, "servidor_ativo", v); err != nil {
		return err
	}
	return s.configs.Definir(ctx, "servidor_porta", strconv.Itoa(porta))
}

// NovaChaveServidor gera outra chave — os aparelhos com a antiga deixam de
// ligar (ex: um aparelho perdido).
func (s *ConfigService) NovaChaveServidor(ctx context.Context) (string, error) {
	const letras = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // sem 0/O nem 1/I
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = letras[int(b[i])%len(letras)]
	}
	chave := string(b[:4]) + "-" + string(b[4:])
	return chave, s.configs.Definir(ctx, "servidor_chave", chave)
}

// DefinirModulo liga ou desliga um módulo numa loja já configurada (ex: o
// modo restaurante, que se liga depois do wizard).
func (s *ConfigService) DefinirModulo(ctx context.Context, m domain.Modulo, ativo bool) error {
	conhecido := false
	for _, x := range domain.TodosModulos {
		conhecido = conhecido || x == m
	}
	if !conhecido {
		return fmt.Errorf("módulo desconhecido: %q", m)
	}
	v := "false"
	if ativo {
		v = "true"
	}
	return s.configs.Definir(ctx, "modulo_"+string(m), v)
}
