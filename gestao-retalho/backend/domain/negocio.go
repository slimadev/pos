package domain

// NegocioID é fixo em 1 porque cada executável cmd/app serve sempre uma
// única loja (SDD secção 2.1) — a linha semeada pela migration core nunca
// muda de identidade, só de conteúdo (o wizard faz UPDATE, nunca INSERT).
const NegocioID int64 = 1

// UtilizadorAdminSeedID é o utilizador de arranque criado pela migration
// core (ver 000001_core_schema.up.sql) — o wizard completa os seus dados
// reais no passo 4 em vez de criar um segundo utilizador.
const UtilizadorAdminSeedID int64 = 1

type TipoNegocio string

const (
	TipoNegocioMercearia     TipoNegocio = "mercearia"
	TipoNegocioFerragem      TipoNegocio = "ferragem"
	TipoNegocioFarmacia      TipoNegocio = "farmacia"
	TipoNegocioBottlestore   TipoNegocio = "bottlestore"
	TipoNegocioPersonalizado TipoNegocio = "personalizado"
)

type Negocio struct {
	ID          int64
	Nome        string
	TipoNegocio TipoNegocio
	Ativo       bool
}

// Modulo é uma funcionalidade opcional que o wizard ativa/desativa (SDD
// secção 9.1/9.2) — o menu dinâmico do frontend (9.5) mostra/esconde ecrãs
// conforme estes valores gravados em `configuracoes`.
type Modulo string

const (
	ModuloPosDireto       Modulo = "pos_direto"
	ModuloContasAbertas   Modulo = "contas_abertas"
	ModuloFichasTecnicas  Modulo = "fichas_tecnicas"
	ModuloLotesValidade   Modulo = "lotes_validade"
	ModuloReceitasMedicas Modulo = "receitas_medicas"
	ModuloVendaGranel     Modulo = "venda_granel"
	ModuloDividas         Modulo = "dividas"
)

// Tema identifica qual ficheiro CSS de tema (frontend/src/themes) aplicar —
// os valores coincidem de propósito com os nomes de tema já existentes no
// frontend desde a Fase 0.
type Tema string

const (
	TemaAdmin       Tema = "admin"
	TemaBottlestore Tema = "bottlestore"
	TemaFerragem    Tema = "ferragem"
	TemaMercearia   Tema = "mercearia"
	TemaFarmacia    Tema = "farmacia"
)

// PresetNegocio descreve um dos quatro tipos de negócio pré-configurados do
// wizard (SDD 9.1) — módulos e tema já vêm decididos; só "Personalizado"
// exige o utilizador escolher módulos manualmente (ver secção 9.2).
type PresetNegocio struct {
	Tipo      TipoNegocio
	Nome      string
	Descricao string
	Modulos   []Modulo
	Tema      Tema
}

// PresetsNegocio é a tabela da secção 9.1 do SDD, uma entrada por tipo.
// "Personalizado" não aparece aqui — não tem módulos pré-decididos, o
// wizard mostra antes o checklist completo (ver ConfigService).
var PresetsNegocio = []PresetNegocio{
	{
		Tipo:      TipoNegocioMercearia,
		Nome:      "Mercearia",
		Descricao: "POS direto e venda a granel por peso",
		Modulos:   []Modulo{ModuloPosDireto, ModuloVendaGranel},
		Tema:      TemaMercearia,
	},
	{
		Tipo:      TipoNegocioFerragem,
		Nome:      "Ferragem / Loja de materiais",
		Descricao: "POS direto com unidades de embalagem",
		Modulos:   []Modulo{ModuloPosDireto},
		Tema:      TemaFerragem,
	},
	{
		Tipo:      TipoNegocioFarmacia,
		Nome:      "Farmácia",
		Descricao: "Lotes com validade (FEFO) e receitas médicas",
		Modulos:   []Modulo{ModuloPosDireto, ModuloLotesValidade, ModuloReceitasMedicas},
		Tema:      TemaFarmacia,
	},
	{
		Tipo:      TipoNegocioBottlestore,
		Nome:      "Bottlestore (bebidas + comida)",
		Descricao: "Contas abertas, dívidas e produtos compostos com ficha técnica",
		Modulos:   []Modulo{ModuloContasAbertas, ModuloDividas, ModuloFichasTecnicas},
		Tema:      TemaBottlestore,
	},
}

// TodosModulos é o checklist completo oferecido em "Personalizado" (SDD 9.2).
var TodosModulos = []Modulo{
	ModuloPosDireto,
	ModuloContasAbertas,
	ModuloFichasTecnicas,
	ModuloLotesValidade,
	ModuloReceitasMedicas,
	ModuloVendaGranel,
	ModuloDividas,
}
