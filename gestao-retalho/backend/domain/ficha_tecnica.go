package domain

// FichaTecnica é uma linha de ingrediente na receita de um produto
// `composto` ou `semi_elaborado` (SDD 3.3) — um produto composto tem N
// linhas, uma por ingrediente. Não confundir com `receitas_medicas` da
// farmácia (Fase 6): esta é ficha técnica de produto, aquela é prescrição
// médica.
type FichaTecnica struct {
	ID                   int64
	ProdutoCompostoID    int64
	ProdutoIngredienteID int64
	Quantidade           float64
	PerdaPercentual      float64
}

// ConsumoReal é quanto sai do stock do ingrediente por unidade do composto
// vendida — sempre >= Quantidade; só é igual quando não há perda.
func (f FichaTecnica) ConsumoReal() float64 {
	if f.PerdaPercentual <= 0 {
		return f.Quantidade
	}
	return f.Quantidade / (1 - f.PerdaPercentual/100)
}
