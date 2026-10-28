package copias_test

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/config"
	"gestao-retalho/backend/copias"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// criarLoja monta na pasta de dados atual o que o primeiro arranque cria:
// admin.db com a loja registada e a base da loja configurada.
func criarLoja(t *testing.T, nome string) {
	t.Helper()
	ctx := context.Background()
	caminhoAdmin, err := config.ResolveAdminDBPath()
	if err != nil {
		t.Fatal(err)
	}
	adb, err := repository.NewAdminSQLiteConn(caminhoAdmin)
	if err != nil {
		t.Fatal(err)
	}
	defer adb.Close()
	caminhoLoja, err := config.ResolveNovaLojaDBPath(nome)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.NewLojaRepository(adb).Criar(ctx, nome, string(caminhoLoja)); err != nil {
		t.Fatal(err)
	}
	app, fechar, err := admin.AbrirLoja(string(caminhoLoja))
	if err != nil {
		t.Fatal(err)
	}
	defer fechar()
	if err := app.FinalizarWizard(usecase.FinalizarWizardParams{
		NegocioNome: nome, TipoNegocio: "ferragem", ArmazemPrincipalNome: "Loja",
		AdminNome: "Ana", AdminUsername: "ana", AdminPassword: "segredo123",
	}); err != nil {
		t.Fatal(err)
	}
}

func usarPasta(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	config.DefinirDiretorioBase(dir)
	t.Cleanup(func() { config.DefinirDiretorioBase("") })
	return dir
}

// O caso que importa: a loja passa de uma máquina para outra (pasta
// diferente, caminhos absolutos diferentes) e abre com os mesmos dados.
func TestCopiaERestauroNoutraMaquina(t *testing.T) {
	usarPasta(t)
	criarLoja(t, "Ferragem Central")

	if _, ok := copias.Ultima(); ok {
		t.Fatal("ainda não devia haver cópias")
	}
	c, err := copias.Criar()
	if err != nil {
		t.Fatalf("Criar: %v", err)
	}
	if ultima, ok := copias.Ultima(); !ok || ultima.IsZero() {
		t.Fatal("Ultima devia devolver a cópia criada")
	}
	conteudo := lerZip(t, c.Caminho)
	for _, quer := range []string{"copia.json", "admin.db", "lojas/ferragem-central.db", "config.json"} {
		if !conteudo[quer] {
			t.Errorf("a cópia não tem %s (tem %v)", quer, conteudo)
		}
	}
	// A cópia sai desta máquina (ex: WhatsApp) e entra noutra.
	transporte := filepath.Join(t.TempDir(), c.Nome)
	copiar(t, c.Caminho, transporte)

	destino := usarPasta(t)
	m, err := copias.PrepararRestauro(transporte)
	if err != nil {
		t.Fatalf("PrepararRestauro: %v", err)
	}
	if len(m.Lojas) != 1 || !copias.HaRestauroPendente() {
		t.Fatalf("manifesto = %+v, pendente = %v", m, copias.HaRestauroPendente())
	}
	aplicado, err := copias.AplicarRestauroPendente()
	if err != nil || !aplicado {
		t.Fatalf("AplicarRestauroPendente = %v, %v", aplicado, err)
	}
	if copias.HaRestauroPendente() {
		t.Error("o restauro continuou pendente depois de aplicado")
	}

	caminhoAdmin, _ := config.ResolveAdminDBPath()
	adb, err := repository.NewAdminSQLiteConn(caminhoAdmin)
	if err != nil {
		t.Fatal(err)
	}
	defer adb.Close()
	loja, existe, err := repository.NewLojaRepository(adb).ObterUnica(context.Background())
	if err != nil || !existe {
		t.Fatalf("loja restaurada: existe=%v err=%v", existe, err)
	}
	if !strings.HasPrefix(loja.CaminhoDB, destino) {
		t.Errorf("caminho da loja = %q, devia estar dentro de %q", loja.CaminhoDB, destino)
	}
	app, fechar, err := admin.AbrirLoja(loja.CaminhoDB)
	if err != nil {
		t.Fatal(err)
	}
	defer fechar()
	if ok, _ := app.EstaConfigurado(); !ok {
		t.Error("a loja restaurada devia estar configurada")
	}
	estado, err := app.ObterEstadoConfig()
	if err != nil || estado.Negocio.Nome != "Ferragem Central" {
		t.Errorf("negócio restaurado = %+v, err=%v", estado.Negocio, err)
	}
}

// Os dados que existiam antes do restauro não se perdem: ficam guardados.
func TestRestauroGuardaOsDadosAnteriores(t *testing.T) {
	dir := usarPasta(t)
	criarLoja(t, "Loja A")
	c, err := copias.Criar()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := copias.PrepararRestauro(c.Caminho); err != nil {
		t.Fatal(err)
	}
	if _, err := copias.AplicarRestauroPendente(); err != nil {
		t.Fatal(err)
	}
	antes, _ := filepath.Glob(filepath.Join(dir, "gestao-retalho-admin", "antes-do-restauro-*", "admin.db"))
	if len(antes) != 1 {
		t.Errorf("devia haver uma cópia dos dados anteriores, há %v", antes)
	}
}

func TestPrepararRestauroRecusaFicheirosInvalidos(t *testing.T) {
	usarPasta(t)
	dir := t.TempDir()

	naoZip := filepath.Join(dir, "x.zip")
	os.WriteFile(naoZip, []byte("isto não é um zip"), 0o644)

	semAdmin := filepath.Join(dir, "sem-admin.zip")
	escreverZip(t, semAdmin, map[string]string{"copia.json": `{"Versao":1}`})

	malicioso := filepath.Join(dir, "slip.zip")
	escreverZip(t, malicioso, map[string]string{"../../fora.db": "x", "copia.json": `{"Versao":1}`})

	danificado := filepath.Join(dir, "danificado.zip")
	escreverZip(t, danificado, map[string]string{"copia.json": `{"Versao":1}`, "admin.db": "não é sqlite"})

	for _, f := range []string{naoZip, semAdmin, malicioso, danificado} {
		if _, err := copias.PrepararRestauro(f); err == nil {
			t.Errorf("%s devia ser recusado", filepath.Base(f))
		}
		if copias.HaRestauroPendente() {
			t.Errorf("%s deixou um restauro pendente", filepath.Base(f))
		}
	}
}

func TestCaminhoDeSoAceitaCopiasGuardadas(t *testing.T) {
	usarPasta(t)
	criarLoja(t, "Loja")
	c, err := copias.Criar()
	if err != nil {
		t.Fatal(err)
	}
	if got, err := copias.CaminhoDe(c.Nome); err != nil || got != c.Caminho {
		t.Errorf("CaminhoDe(%q) = %q, %v", c.Nome, got, err)
	}
	for _, mau := range []string{"../admin.db", "admin.db", "copia-x.zip", `..\x.zip`} {
		if _, err := copias.CaminhoDe(mau); err == nil {
			t.Errorf("CaminhoDe(%q) devia falhar", mau)
		}
	}
}

// As cópias novas têm extensão própria (o gestor de ficheiros do Android
// abria os .zip como pastas); as .zip de versões anteriores continuam
// listadas e aceites.
func TestExtensaoPropriaEAceitaCopiasAntigas(t *testing.T) {
	usarPasta(t)
	criarLoja(t, "Loja")
	c, err := copias.Criar()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(c.Nome, copias.Extensao) {
		t.Errorf("nome = %q, devia terminar em %s", c.Nome, copias.Extensao)
	}
	antiga := filepath.Join(filepath.Dir(c.Caminho), "copia-20250101-120000.zip")
	copiar(t, c.Caminho, antiga)

	lista, err := copias.Listar()
	if err != nil || len(lista) != 2 {
		t.Fatalf("Listar = %d cópias, %v (want 2)", len(lista), err)
	}
	if _, err := copias.CaminhoDe(filepath.Base(antiga)); err != nil {
		t.Errorf("cópia .zip antiga recusada: %v", err)
	}
	if _, err := copias.PrepararRestauro(antiga); err != nil {
		t.Errorf("restaurar .zip antiga: %v", err)
	}
}

func lerZip(t *testing.T, caminho string) map[string]bool {
	t.Helper()
	r, err := zip.OpenReader(caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	out := map[string]bool{}
	for _, f := range r.File {
		out[f.Name] = true
	}
	return out
}

func escreverZip(t *testing.T, caminho string, entradas map[string]string) {
	t.Helper()
	f, err := os.Create(caminho)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for n, c := range entradas {
		e, _ := w.Create(n)
		e.Write([]byte(c))
	}
	w.Close()
	f.Close()
}

func copiar(t *testing.T, de, para string) {
	t.Helper()
	b, err := os.ReadFile(de)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(para, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
