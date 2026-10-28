package admin

import (
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"gestao-retalho/backend/copias"
	"gestao-retalho/backend/export"
)

// --- Cópias de segurança -------------------------------------------------
//
// No terminal Android os dados existem só lá: a cópia tem de sair dele
// (partilhar para Drive/WhatsApp/email/PC). No desktop fica na pasta de
// dados e pode copiar-se para uma pen.

// diasAvisoCopia: a partir de quantos dias sem cópia o dashboard avisa.
const diasAvisoCopia = 7

type EstadoCopias struct {
	// UltimaEm é a data da cópia mais recente; nil se nunca se fez nenhuma.
	UltimaEm *time.Time
	// DiasDesdeUltima é -1 quando nunca se fez nenhuma.
	DiasDesdeUltima  int
	Copias           []copias.Copia
	RestauroPendente bool
}

func (a *AdminApp) EstadoCopiasSeguranca() (EstadoCopias, error) {
	if err := a.exigirSessao(); err != nil {
		return EstadoCopias{}, err
	}
	lista, err := copias.Listar()
	if err != nil {
		return EstadoCopias{}, err
	}
	e := EstadoCopias{DiasDesdeUltima: -1, Copias: lista, RestauroPendente: copias.HaRestauroPendente()}
	if len(lista) > 0 {
		u := lista[0].CriadaEm
		e.UltimaEm = &u
		e.DiasDesdeUltima = int(time.Since(u).Hours() / 24)
	}
	return e, nil
}

// CriarCopiaSeguranca grava uma cópia nova de todos os dados.
func (a *AdminApp) CriarCopiaSeguranca() (copias.Copia, error) {
	if err := a.exigirSessao(); err != nil {
		return copias.Copia{}, err
	}
	return copias.Criar()
}

// PartilharCopiaSeguranca abre a cópia `nome` fora da app: no Android, a
// folha de partilha; no desktop, o programa associado a .zip.
func (a *AdminApp) PartilharCopiaSeguranca(nome string) error {
	if err := a.exigirSessao(); err != nil {
		return err
	}
	caminho, err := copias.CaminhoDe(nome)
	if err != nil {
		return err
	}
	return export.AbrirFicheiro(caminho)
}

// exigirSessaoOuTerminalNovo deixa restaurar sem login enquanto não existir
// nenhum administrador: num terminal novo (ou reposto de fábrica) o ecrã de
// primeiro arranque tem de poder trazer a loja de volta — e não há ainda
// dados nenhuns a proteger. Depois disso, só com sessão, como o resto.
func (a *AdminApp) exigirSessaoOuTerminalNovo() error {
	if a.exigirSessao() == nil {
		return nil
	}
	if novo, err := a.Auth.PrecisaConfigurar(a.ctx); err == nil && novo {
		return nil
	}
	return a.exigirSessao()
}

// EscolherCopiaParaRestaurar abre o seletor de ficheiros para um .zip de
// cópia; devolve "" se o utilizador cancelar.
func (a *AdminApp) EscolherCopiaParaRestaurar() (string, error) {
	if err := a.exigirSessaoOuTerminalNovo(); err != nil {
		return "", err
	}
	if escolhedorFicheiros != nil {
		return escolhedorFicheiros("Escolher a cópia de segurança a restaurar", "retalho;zip")
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Escolher a cópia de segurança a restaurar",
		Filters: []runtime.FileFilter{
			{DisplayName: "Cópia de segurança (*.retalho, *.zip)", Pattern: "*.retalho;*.zip"},
		},
	})
}

// PrepararRestauroCopia valida a cópia e deixa-a pronta: os dados só são
// trocados quando a aplicação voltar a arrancar.
func (a *AdminApp) PrepararRestauroCopia(caminho string) (copias.Manifesto, error) {
	if err := a.exigirSessaoOuTerminalNovo(); err != nil {
		return copias.Manifesto{}, err
	}
	return copias.PrepararRestauro(caminho)
}
