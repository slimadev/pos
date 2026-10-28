// Package copias faz as cópias de segurança de todos os dados do sistema
// (admin.db, as bases das lojas e o config.json) para um único .zip, e o
// restauro de uma cópia. Num terminal Android os dados existem só lá: sem
// cópias, um terminal roubado, avariado ou reposto leva a loja inteira.
//
// O restauro faz-se em dois tempos, para nunca substituir bases abertas:
// PrepararRestauro valida o .zip e deixa-o extraído numa pasta pendente;
// AplicarRestauroPendente troca os ficheiros no arranque seguinte, antes de
// qualquer base ser aberta.
package copias

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"gestao-retalho/backend/config"
)

// Extensao das cópias. Por dentro é um .zip, mas com extensão própria: o
// gestor de ficheiros do Android 7 (Sunmi) abria os .zip como pastas no
// seletor, em vez de os deixar escolher para restaurar.
const Extensao = ".retalho"

const (
	pastaCopias   = "copias"
	pastaPendente = "restauro-pendente"
	marcaPronto   = ".pronto"
	manifesto     = "copia.json"
	// Quantas cópias ficam guardadas na própria máquina; as mais antigas
	// apagam-se. A cópia que conta é a que sai dela (partilhada).
	copiasGuardadas = 10
)

// Copia é uma cópia de segurança guardada localmente.
type Copia struct {
	Nome     string
	Caminho  string
	Tamanho  int64
	CriadaEm time.Time
}

// Manifesto descreve o conteúdo de uma cópia (copia.json dentro do .zip).
type Manifesto struct {
	Versao   int
	CriadaEm time.Time
	Lojas    []string
}

type pastas struct{ admin, app, lojas, copias string }

func resolverPastas() (pastas, error) {
	dirAdmin, dirApp, err := config.DiretoriosDados()
	if err != nil {
		return pastas{}, err
	}
	return pastas{
		admin:  dirAdmin,
		app:    dirApp,
		lojas:  filepath.Join(dirAdmin, "lojas"),
		copias: filepath.Join(dirAdmin, pastaCopias),
	}, nil
}

// Criar grava uma cópia nova com todos os dados e devolve-a. As bases são
// copiadas com VACUUM INTO — uma fotografia consistente mesmo com a app em
// uso (vendas a meio não ficam a meio na cópia).
func Criar() (Copia, error) {
	p, err := resolverPastas()
	if err != nil {
		return Copia{}, err
	}
	adminDB := filepath.Join(p.admin, "admin.db")
	if _, err := os.Stat(adminDB); err != nil {
		return Copia{}, errors.New("ainda não há dados para copiar")
	}
	if err := os.MkdirAll(p.copias, 0o755); err != nil {
		return Copia{}, err
	}
	trabalho, err := os.MkdirTemp(p.copias, ".a-criar-")
	if err != nil {
		return Copia{}, err
	}
	defer os.RemoveAll(trabalho)

	agora := time.Now()
	m := Manifesto{Versao: 1, CriadaEm: agora}
	ficheiros := map[string]string{} // nome no zip → ficheiro local

	if err := fotografar(adminDB, filepath.Join(trabalho, "admin.db")); err != nil {
		return Copia{}, fmt.Errorf("copiar admin.db: %w", err)
	}
	ficheiros["admin.db"] = filepath.Join(trabalho, "admin.db")

	lojas, _ := filepath.Glob(filepath.Join(p.lojas, "*.db"))
	for _, l := range lojas {
		nome := filepath.Base(l)
		dst := filepath.Join(trabalho, "loja-"+nome)
		if err := fotografar(l, dst); err != nil {
			return Copia{}, fmt.Errorf("copiar %s: %w", nome, err)
		}
		ficheiros["lojas/"+nome] = dst
		m.Lojas = append(m.Lojas, nome)
	}
	if cfg := filepath.Join(p.app, "config.json"); existe(cfg) {
		ficheiros["config.json"] = cfg
	}

	nome := "copia-" + agora.Format("20060102-150405") + Extensao
	destino := filepath.Join(p.copias, nome)
	if err := escreverZip(destino, ficheiros, m); err != nil {
		return Copia{}, err
	}
	podar(p.copias)

	info, err := os.Stat(destino)
	if err != nil {
		return Copia{}, err
	}
	return Copia{Nome: nome, Caminho: destino, Tamanho: info.Size(), CriadaEm: agora}, nil
}

// Listar devolve as cópias guardadas nesta máquina, a mais recente primeiro.
func Listar() ([]Copia, error) {
	p, err := resolverPastas()
	if err != nil {
		return nil, err
	}
	zips := listarFicheiros(p.copias)
	out := make([]Copia, 0, len(zips))
	for _, z := range zips {
		info, err := os.Stat(z)
		if err != nil {
			continue
		}
		out = append(out, Copia{Nome: filepath.Base(z), Caminho: z, Tamanho: info.Size(), CriadaEm: info.ModTime()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CriadaEm.After(out[j].CriadaEm) })
	return out, nil
}

// Ultima devolve a data da cópia mais recente (false se não houver).
func Ultima() (time.Time, bool) {
	l, err := Listar()
	if err != nil || len(l) == 0 {
		return time.Time{}, false
	}
	return l[0].CriadaEm, true
}

// CaminhoDe devolve o caminho de uma cópia guardada pelo nome, recusando
// tudo o que não seja um ficheiro da pasta de cópias.
func CaminhoDe(nome string) (string, error) {
	p, err := resolverPastas()
	if err != nil {
		return "", err
	}
	if nome != filepath.Base(nome) || !strings.HasPrefix(nome, "copia-") ||
		!(strings.HasSuffix(nome, Extensao) || strings.HasSuffix(nome, ".zip")) {
		return "", fmt.Errorf("cópia inválida: %q", nome)
	}
	c := filepath.Join(p.copias, nome)
	if !existe(c) {
		return "", fmt.Errorf("cópia não encontrada: %q", nome)
	}
	return c, nil
}

// PrepararRestauro valida a cópia em `caminhoZip` e deixa-a pronta a
// aplicar no próximo arranque. Não toca nos dados atuais.
func PrepararRestauro(caminhoZip string) (Manifesto, error) {
	p, err := resolverPastas()
	if err != nil {
		return Manifesto{}, err
	}
	r, err := zip.OpenReader(caminhoZip)
	if err != nil {
		return Manifesto{}, fmt.Errorf("o ficheiro não é uma cópia de segurança válida: %w", err)
	}
	defer r.Close()

	pendente := filepath.Join(p.admin, pastaPendente)
	if err := os.RemoveAll(pendente); err != nil {
		return Manifesto{}, err
	}
	if err := os.MkdirAll(filepath.Join(pendente, "lojas"), 0o755); err != nil {
		return Manifesto{}, err
	}
	falhar := func(e error) (Manifesto, error) {
		os.RemoveAll(pendente)
		return Manifesto{}, e
	}

	var m Manifesto
	temAdmin, temManifesto := false, false
	for _, f := range r.File {
		nome := f.Name
		// Nada fora da pasta pendente (zip slip) nem ficheiros desconhecidos.
		if strings.Contains(nome, "..") || path.IsAbs(nome) || strings.Contains(nome, "\\") {
			return falhar(fmt.Errorf("cópia inválida: entrada %q", nome))
		}
		switch {
		case nome == manifesto:
			if err := lerJSON(f, &m); err != nil {
				return falhar(fmt.Errorf("cópia inválida: %w", err))
			}
			temManifesto = true
			continue
		case nome == "admin.db":
			temAdmin = true
		case nome == "config.json":
		case strings.HasPrefix(nome, "lojas/") && strings.HasSuffix(nome, ".db") && path.Base(nome) == strings.TrimPrefix(nome, "lojas/"):
		default:
			continue
		}
		if err := extrair(f, filepath.Join(pendente, filepath.FromSlash(nome))); err != nil {
			return falhar(err)
		}
	}
	if !temAdmin || !temManifesto {
		return falhar(errors.New("cópia inválida: faltam ficheiros (não é uma cópia desta aplicação)"))
	}

	bases, _ := filepath.Glob(filepath.Join(pendente, "lojas", "*.db"))
	bases = append(bases, filepath.Join(pendente, "admin.db"))
	for _, b := range bases {
		if err := verificarIntegridade(b); err != nil {
			return falhar(fmt.Errorf("cópia danificada (%s): %w", filepath.Base(b), err))
		}
	}
	if err := os.WriteFile(filepath.Join(pendente, marcaPronto), []byte(time.Now().Format(time.RFC3339)), 0o644); err != nil {
		return falhar(err)
	}
	return m, nil
}

// HaRestauroPendente diz se há uma cópia à espera do próximo arranque.
func HaRestauroPendente() bool {
	p, err := resolverPastas()
	if err != nil {
		return false
	}
	return existe(filepath.Join(p.admin, pastaPendente, marcaPronto))
}

// AplicarRestauroPendente troca os dados atuais pelos da cópia preparada
// (se houver). Chama-se no arranque, antes de abrir qualquer base. Os dados
// substituídos ficam em antes-do-restauro-<data>/, por segurança.
func AplicarRestauroPendente() (aplicado bool, err error) {
	p, err := resolverPastas()
	if err != nil {
		return false, err
	}
	pendente := filepath.Join(p.admin, pastaPendente)
	if !existe(filepath.Join(pendente, marcaPronto)) {
		return false, nil
	}

	antes := filepath.Join(p.admin, "antes-do-restauro-"+time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(antes, 0o755); err != nil {
		return false, err
	}
	// Bases atuais (com -wal/-shm: um WAL antigo aplicado à base nova
	// corrompia-a), lojas e config.json saem para `antes`.
	for _, f := range []string{"admin.db", "admin.db-wal", "admin.db-shm"} {
		if err := moverSeExistir(filepath.Join(p.admin, f), filepath.Join(antes, f)); err != nil {
			return false, err
		}
	}
	if err := moverSeExistir(p.lojas, filepath.Join(antes, "lojas")); err != nil {
		return false, err
	}
	if err := moverSeExistir(filepath.Join(p.app, "config.json"), filepath.Join(antes, "config.json")); err != nil {
		return false, err
	}

	if err := os.Rename(filepath.Join(pendente, "admin.db"), filepath.Join(p.admin, "admin.db")); err != nil {
		return false, err
	}
	if err := os.Rename(filepath.Join(pendente, "lojas"), p.lojas); err != nil {
		return false, err
	}
	if existe(filepath.Join(pendente, "config.json")) {
		if err := os.MkdirAll(p.app, 0o755); err != nil {
			return false, err
		}
		if err := os.Rename(filepath.Join(pendente, "config.json"), filepath.Join(p.app, "config.json")); err != nil {
			return false, err
		}
	}
	if err := apontarLojasPara(filepath.Join(p.admin, "admin.db"), p.lojas); err != nil {
		return false, err
	}
	return true, os.RemoveAll(pendente)
}

// apontarLojasPara corrige o registo das lojas: guarda caminhos absolutos,
// e a cópia pode vir de outra máquina (ex: do terminal para o PC).
func apontarLojasPara(adminDB, dirLojas string) error {
	db, err := sql.Open("sqlite", adminDB)
	if err != nil {
		return err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT id, caminho_db FROM lojas`)
	if err != nil {
		return err
	}
	type loja struct {
		id      int64
		caminho string
	}
	var lojas []loja
	for rows.Next() {
		var l loja
		if err := rows.Scan(&l.id, &l.caminho); err != nil {
			rows.Close()
			return err
		}
		lojas = append(lojas, l)
	}
	rows.Close()
	for _, l := range lojas {
		// filepath.Base não percebe "\" fora do Windows: a cópia pode vir de lá.
		nome := l.caminho[strings.LastIndexAny(l.caminho, `/\`)+1:]
		if _, err := db.Exec(`UPDATE lojas SET caminho_db = ? WHERE id = ?`, filepath.Join(dirLojas, nome), l.id); err != nil {
			return err
		}
	}
	return nil
}

// --- utilitários ------------------------------------------------------------

func fotografar(origem, destino string) error {
	db, err := sql.Open("sqlite", origem)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`VACUUM INTO ?`, destino)
	return err
}

func verificarIntegridade(caminho string) error {
	db, err := sql.Open("sqlite", caminho)
	if err != nil {
		return err
	}
	defer db.Close()
	var r string
	if err := db.QueryRow(`PRAGMA integrity_check`).Scan(&r); err != nil {
		return err
	}
	if r != "ok" {
		return errors.New(r)
	}
	return nil
}

func escreverZip(destino string, ficheiros map[string]string, m Manifesto) error {
	tmp := destino + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := zip.NewWriter(f)
	escrever := func() error {
		jm, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return err
		}
		e, err := w.Create(manifesto)
		if err != nil {
			return err
		}
		if _, err := e.Write(jm); err != nil {
			return err
		}
		nomes := make([]string, 0, len(ficheiros))
		for n := range ficheiros {
			nomes = append(nomes, n)
		}
		sort.Strings(nomes)
		for _, n := range nomes {
			e, err := w.Create(n)
			if err != nil {
				return err
			}
			src, err := os.Open(ficheiros[n])
			if err != nil {
				return err
			}
			_, err = io.Copy(e, src)
			src.Close()
			if err != nil {
				return err
			}
		}
		return w.Close()
	}
	if err := escrever(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, destino)
}

// listarFicheiros devolve as cópias da pasta: as atuais (.retalho) e as de
// versões anteriores (.zip).
func listarFicheiros(dir string) []string {
	atuais, _ := filepath.Glob(filepath.Join(dir, "copia-*"+Extensao))
	antigas, _ := filepath.Glob(filepath.Join(dir, "copia-*.zip"))
	return append(atuais, antigas...)
}

func podar(dir string) {
	zips := listarFicheiros(dir)
	// O nome tem a data: ordenar pelo nome sem extensão = ordem cronológica.
	sort.Slice(zips, func(i, j int) bool {
		return strings.TrimSuffix(filepath.Base(zips[i]), filepath.Ext(zips[i])) <
			strings.TrimSuffix(filepath.Base(zips[j]), filepath.Ext(zips[j]))
	})
	for len(zips) > copiasGuardadas {
		os.Remove(zips[0])
		zips = zips[1:]
	}
}

func lerJSON(f *zip.File, v any) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	return json.NewDecoder(rc).Decode(v)
}

func extrair(f *zip.File, destino string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.Create(destino)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func moverSeExistir(de, para string) error {
	if !existe(de) {
		return nil
	}
	return os.Rename(de, para)
}

func existe(caminho string) bool {
	_, err := os.Stat(caminho)
	return err == nil
}
