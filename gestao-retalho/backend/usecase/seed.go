package usecase

import (
	"context"
	"fmt"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

type seedProduto struct {
	Nome        string
	NomeUnidade string
	PrecoCusto  float64
	PrecoVenda  float64
}

type seedCategoria struct {
	Nome     string
	Produtos []seedProduto
}

// seedsPorTipo dá aos quatro tipos pré-configurados um ponto de partida
// reconhecível (SDD 9.2: "aplica seed de categorias/produtos exemplo do
// tipo escolhido") — produtos reais que o operador pode editar ou apagar,
// não dados de demonstração fake presos ao ecrã.
var seedsPorTipo = map[domain.TipoNegocio][]seedCategoria{
	domain.TipoNegocioMercearia: {
		{Nome: "Mercearia", Produtos: []seedProduto{
			{"Arroz 5kg", "unidade", 250, 320},
			{"Óleo 1L", "unidade", 90, 130},
			{"Açúcar 1kg", "unidade", 55, 80},
		}},
		{Nome: "Bebidas", Produtos: []seedProduto{
			{"Água 500ml", "unidade", 25, 60},
			{"Coca-Cola 500ml", "unidade", 70, 120},
		}},
	},
	domain.TipoNegocioFerragem: {
		{Nome: "Cimento e areia", Produtos: []seedProduto{
			{"Cimento 50kg", "saco", 480, 620},
		}},
		{Nome: "Elétrico", Produtos: []seedProduto{
			{"Cabo 2,5mm²", "metro", 60, 95},
			{"Lâmpada LED 9W", "unidade", 90, 130},
		}},
		{Nome: "Ferramentas", Produtos: []seedProduto{
			{"Chave inglesa 10\"", "unidade", 650, 890},
		}},
	},
	domain.TipoNegocioFarmacia: {
		{Nome: "Medicamentos", Produtos: []seedProduto{
			{"Paracetamol 500mg", "caixa", 30, 55},
		}},
		{Nome: "Higiene", Produtos: []seedProduto{
			{"Álcool gel 250ml", "unidade", 70, 110},
		}},
	},
	domain.TipoNegocioBottlestore: {
		{Nome: "Cervejas", Produtos: []seedProduto{
			{"2M 550ml", "unidade", 130, 180},
			{"Laurentina Preta 550ml", "unidade", 140, 195},
		}},
		{Nome: "Refrigerantes", Produtos: []seedProduto{
			{"Coca-Cola 500ml", "unidade", 70, 120},
			{"Água 500ml", "unidade", 25, 60},
		}},
	},
}

// aplicarSeed cria as categorias/produtos de exemplo do tipo escolhido.
// Personalizado (e qualquer tipo sem entrada na tabela) fica sem seed —
// nada de fabricar módulos "porque sim" para um negócio que o utilizador
// disse explicitamente que ia configurar à mão.
func aplicarSeed(ctx context.Context, tipo domain.TipoNegocio, categorias *repository.CategoriaRepository, produtos *ProdutoService) error {
	for _, cat := range seedsPorTipo[tipo] {
		categoria, err := categorias.Criar(ctx, domain.Categoria{NegocioID: domain.NegocioID, Nome: cat.Nome})
		if err != nil {
			return fmt.Errorf("seed categoria %q: %w", cat.Nome, err)
		}
		for _, p := range cat.Produtos {
			categoriaID := categoria.ID
			_, _, err := produtos.Criar(ctx, NovoProduto{
				Produto: domain.Produto{
					NegocioID:   domain.NegocioID,
					CategoriaID: &categoriaID,
					Nome:        p.Nome,
					TipoProduto: domain.TipoProdutoSimples,
					UnidadeBase: p.NomeUnidade,
				},
				Unidades: []domain.UnidadeProduto{{
					NomeUnidade:      p.NomeUnidade,
					QtdEmUnidadeBase: 1,
					PrecoCusto:       p.PrecoCusto,
					PrecoVenda:       p.PrecoVenda,
					IsPadraoVenda:    true,
					IsPadraoCompra:   true,
				}},
			})
			if err != nil {
				return fmt.Errorf("seed produto %q: %w", p.Nome, err)
			}
		}
	}
	return nil
}
