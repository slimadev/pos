// Package gobridge é o backend Go da app Android, exportado com gomobile
// (`gomobile bind`) para o Kotlin. Faz o papel que o Wails faz no desktop:
// os frontends chamam window.go.admin.AdminApp.X(...) e window.go.api.App.X(...)
// e a ponte JS entrega cada chamada a Chamar(alvo, método, argumentos JSON).
//
// A API exportada é mínima de propósito (só strings, erros e uma interface),
// porque é o que o gomobile sabe traduzir para Java/Kotlin.
package gobridge

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	_ "time/tzdata" // fusos horários embutidos: não depende da base do sistema

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/api"
	"gestao-retalho/backend/config"
	"gestao-retalho/backend/copias"
	"gestao-retalho/backend/export"
	"gestao-retalho/backend/repository"
)

// Plataforma é implementada em Kotlin: o que no desktop o sistema operativo
// ou o Wails fazem e no Android tem de ser a app nativa a fazer.
type Plataforma interface {
	// AbrirFicheiro mostra um PDF/Excel gerado (visualizador ou partilha).
	AbrirFicheiro(caminho string) error
	// EscolherFicheiro abre o seletor do Android e devolve o caminho de uma
	// cópia local do ficheiro escolhido, ou "" se o utilizador cancelar.
	// `extensoes` vem como "xlsx;xlsm".
	EscolherFicheiro(titulo, extensoes string) (string, error)
}

var (
	mu       sync.Mutex
	adminApp *admin.AdminApp
	lojas    *repository.LojaRepository
	pdvApp   *api.App
)

// Iniciar arranca o backend com os dados em `dirDados` (a pasta privada da
// app). `fusoHorario` é o do terminal (ex: "Africa/Maputo"): o Go no Android
// não o lê do sistema e ficava em UTC — as vendas sairiam com a hora errada
// e "hoje" nos relatórios mudaria de dia às 22h. Chama-se uma vez, antes de
// qualquer Chamar; chamadas seguintes não fazem nada.
func Iniciar(dirDados, fusoHorario string, p Plataforma) error {
	mu.Lock()
	defer mu.Unlock()
	if adminApp != nil {
		return nil
	}
	if dirDados == "" {
		return errors.New("pasta de dados vazia")
	}
	if fusoHorario != "" {
		loc, err := time.LoadLocation(fusoHorario)
		if err != nil {
			return fmt.Errorf("fuso horário %q: %w", fusoHorario, err)
		}
		time.Local = loc
		if err := definirFusoDoSQLite(tzPosix(loc, time.Now())); err != nil {
			return fmt.Errorf("fuso horário do SQLite: %w", err)
		}
	}

	config.DefinirDiretorioBase(dirDados)
	// Restauro de cópia escolhido antes de reiniciar: troca os dados agora,
	// antes de qualquer base estar aberta.
	if _, err := copias.AplicarRestauroPendente(); err != nil {
		return fmt.Errorf("restaurar cópia de segurança: %w", err)
	}
	if p != nil {
		export.DefinirAbridor(p.AbrirFicheiro)
		admin.DefinirEscolhedorFicheiros(p.EscolherFicheiro)
	}

	var app *admin.AdminApp
	var repo *repository.LojaRepository
	container := fx.New(
		fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger }),
		admin.Module,
		fx.Provide(config.ResolveAdminDBPath),
		fx.Populate(&app, &repo),
	)
	if err := container.Err(); err != nil {
		return fmt.Errorf("arrancar backend: %w", err)
	}
	app.OnStartup(context.Background())
	adminApp, lojas = app, repo
	return nil
}

// Chamar executa `metodo` em `alvo` ("admin.AdminApp" ou "api.App") com os
// argumentos em JSON (um array) e devolve o 1.º valor de retorno em JSON.
// Um erro Go devolve-se como erro (a ponte JS rejeita a Promise com a
// mensagem), como no Wails.
func Chamar(alvo, metodo, argsJSON string) (string, error) {
	switch alvo {
	case "admin.AdminApp":
		mu.Lock()
		a := adminApp
		mu.Unlock()
		if a == nil {
			return "", errors.New("backend não iniciado")
		}
		return despachar(a, metodo, argsJSON)
	case "api.App":
		p, err := appDaLoja()
		if err != nil {
			return "", err
		}
		return despachar(p, metodo, argsJSON)
	default:
		return "", fmt.Errorf("alvo desconhecido: %q", alvo)
	}
}

// appDaLoja abre (uma vez) o api.App da única loja deste terminal — o mesmo
// que o cmd/app do desktop abre quando o admin lança o PDV. Só existe depois
// do primeiro arranque (o admin cria a loja).
func appDaLoja() (*api.App, error) {
	mu.Lock()
	defer mu.Unlock()
	if pdvApp != nil {
		return pdvApp, nil
	}
	if lojas == nil {
		return nil, errors.New("backend não iniciado")
	}
	loja, existe, err := lojas.ObterUnica(context.Background())
	if err != nil {
		return nil, err
	}
	if !existe {
		return nil, errors.New("nenhuma loja configurada — conclua a configuração na administração")
	}
	app, _, err := admin.AbrirLoja(loja.CaminhoDB)
	if err != nil {
		return nil, err
	}
	pdvApp = app
	return pdvApp, nil
}
