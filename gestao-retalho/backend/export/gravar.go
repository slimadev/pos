package export

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gestao-retalho/backend/config"
)

var caracteresInvalidosFicheiro = regexp.MustCompile(`[^a-zA-Z0-9-]+`)

// GravarTabela gera a tabela em "xlsx" ou "pdf" na pasta de relatórios e
// devolve o caminho. Não abre o ficheiro — isso precisa do contexto Wails
// de quem chama (cmd/app ou cmd/admin).
func GravarTabela(t Tabela, formato string) (string, error) {
	var gerar func(Tabela) ([]byte, error)
	switch formato {
	case "xlsx":
		gerar = GerarTabelaExcel
	case "pdf":
		gerar = GerarTabelaPDF
	default:
		return "", fmt.Errorf("formato de exportação desconhecido: %q", formato)
	}
	conteudo, err := gerar(t)
	if err != nil {
		return "", err
	}

	dir, err := config.ResolveRelatoriosDir()
	if err != nil {
		return "", err
	}
	nome := strings.Trim(caracteresInvalidosFicheiro.ReplaceAllString(t.Titulo, "-"), "-")
	if nome == "" {
		nome = "relatorio"
	}
	caminho := filepath.Join(dir, fmt.Sprintf("%s-%d.%s", nome, time.Now().Unix(), formato))
	if err := os.WriteFile(caminho, conteudo, 0o644); err != nil {
		return "", fmt.Errorf("gravar relatório: %w", err)
	}
	return caminho, nil
}
