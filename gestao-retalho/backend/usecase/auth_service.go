package usecase

import (
	"context"
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// ErrCredenciaisInvalidas cobre tanto "utilizador não existe" como "password
// errada" com a mesma mensagem — distinguir os dois ao cliente ajudaria
// alguém a adivinhar usernames válidos por tentativa e erro.
var ErrCredenciaisInvalidas = errors.New("utilizador ou password incorretos")

// AuthService fecha a lacuna deixada em aberto desde a Fase 2: o wizard já
// criava um utilizador administrador real (nome/username/password com
// bcrypt), mas não existia nenhum mecanismo de sessão/login a validar essas
// credenciais — api.App usava sempre o mesmo ID fixo.
type AuthService struct {
	utilizadores *repository.UtilizadorRepository
}

func NewAuthService(repo *repository.UtilizadorRepository) *AuthService {
	return &AuthService{utilizadores: repo}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (domain.Utilizador, error) {
	if username == "" || password == "" {
		return domain.Utilizador{}, ErrCredenciaisInvalidas
	}

	u, err := s.utilizadores.ObterPorUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Utilizador{}, ErrCredenciaisInvalidas
	}
	if err != nil {
		return domain.Utilizador{}, err
	}
	if !u.Ativo {
		return domain.Utilizador{}, ErrCredenciaisInvalidas
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return domain.Utilizador{}, ErrCredenciaisInvalidas
	}

	return u, nil
}
