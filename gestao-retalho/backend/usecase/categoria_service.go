package usecase

import (
	"context"
	"fmt"
	"strings"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// CategoriaService faltava por completo até agora — o repositório já
// existia desde a Fase 1 (usado internamente pelo wizard para semear
// categorias por preset, ConfigService), mas nunca teve usecase/binding
// próprios para o operador gerir categorias à mão.
type CategoriaService struct {
	repo *repository.CategoriaRepository
}

func NewCategoriaService(repo *repository.CategoriaRepository) *CategoriaService {
	return &CategoriaService{repo: repo}
}

func (s *CategoriaService) Listar(ctx context.Context, negocioID int64) ([]domain.Categoria, error) {
	return s.repo.Listar(ctx, negocioID)
}

func (s *CategoriaService) Criar(ctx context.Context, c domain.Categoria) (domain.Categoria, error) {
	c.Nome = strings.TrimSpace(c.Nome)
	if c.Nome == "" {
		return domain.Categoria{}, fmt.Errorf("nome da categoria é obrigatório")
	}
	if err := s.nomeLivre(ctx, c.Nome, 0); err != nil {
		return domain.Categoria{}, err
	}
	// Sempre a mesma loja (SDD 2.1) — forçado aqui, não confiado a cada
	// chamador, para nunca haver uma categoria órfã por esquecimento.
	c.NegocioID = domain.NegocioID
	c.Ativo = true
	return s.repo.Criar(ctx, c)
}

func (s *CategoriaService) Atualizar(ctx context.Context, c domain.Categoria) error {
	c.Nome = strings.TrimSpace(c.Nome)
	if c.Nome == "" {
		return fmt.Errorf("nome da categoria é obrigatório")
	}
	if err := s.nomeLivre(ctx, c.Nome, c.ID); err != nil {
		return err
	}
	return s.repo.Atualizar(ctx, c)
}

// nomeLivre recusa um nome que já existe noutra categoria — duas
// "Elétrico" apareciam como dois separadores iguais no PDV.
func (s *CategoriaService) nomeLivre(ctx context.Context, nome string, excetoID int64) error {
	existentes, err := s.repo.Listar(ctx, domain.NegocioID)
	if err != nil {
		return err
	}
	for _, c := range existentes {
		if c.ID != excetoID && mesmoNome(c.Nome, nome) {
			return fmt.Errorf("já existe a categoria %q", c.Nome)
		}
	}
	return nil
}

// Ativar recusa reativar uma categoria cujo nome já está noutra ativa —
// senão desfazia um "Juntar" e voltava a haver duas iguais no PDV.
func (s *CategoriaService) Ativar(ctx context.Context, id int64) error {
	todas, err := s.repo.Listar(ctx, domain.NegocioID)
	if err != nil {
		return err
	}
	for _, c := range todas {
		if c.ID != id {
			continue
		}
		for _, o := range todas {
			if o.ID != id && o.Ativo && mesmoNome(o.Nome, c.Nome) {
				return fmt.Errorf("já existe a categoria %q ativa — mude o nome antes de ativar esta", o.Nome)
			}
		}
	}
	return s.repo.DefinirAtiva(ctx, id, true)
}

func (s *CategoriaService) Desativar(ctx context.Context, id int64) error {
	return s.repo.DefinirAtiva(ctx, id, false)
}

// Juntar acaba com uma categoria repetida (ex: duas "Elétrico" criadas
// antes de os nomes repetidos serem recusados): os produtos e as
// subcategorias de origem passam para destino, e origem fica desativada.
// Devolve quantos produtos mudaram de categoria.
func (s *CategoriaService) Juntar(ctx context.Context, origemID, destinoID int64) (int64, error) {
	if origemID == destinoID {
		return 0, fmt.Errorf("escolha outra categoria para juntar")
	}
	todas, err := s.repo.Listar(ctx, domain.NegocioID)
	if err != nil {
		return 0, err
	}
	var origem, destino *domain.Categoria
	for i := range todas {
		switch todas[i].ID {
		case origemID:
			origem = &todas[i]
		case destinoID:
			destino = &todas[i]
		}
	}
	if origem == nil || destino == nil {
		return 0, fmt.Errorf("categoria não encontrada")
	}
	if !destino.Ativo {
		return 0, fmt.Errorf("a categoria %q está desativada — ative-a antes de juntar", destino.Nome)
	}
	// Se destino era subcategoria de origem, sobe para o lugar de origem;
	// senão ficaria pendurada numa categoria desativada.
	if destino.CategoriaPaiID != nil && *destino.CategoriaPaiID == origem.ID {
		destino.CategoriaPaiID = origem.CategoriaPaiID
		if err := s.repo.Atualizar(ctx, *destino); err != nil {
			return 0, err
		}
	}
	return s.repo.Juntar(ctx, origemID, destinoID)
}
