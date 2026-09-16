package domain

// Categoria suporta subcategorias via CategoriaPaiID opcional (SDD 3.2).
type Categoria struct {
	ID             int64
	NegocioID      int64
	Nome           string
	CategoriaPaiID *int64
}
