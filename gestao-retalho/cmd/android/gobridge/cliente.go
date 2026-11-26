package gobridge

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// portaServidorPadrao: a mesma de usecase.PortaServidorPadrao (o PC do balcão).
const portaServidorPadrao = "8765"

// ligacao é o "modo garçom": o aparelho não usa a base local, fala com o
// PDV do computador do balcão (backend/servidor) pela rede da loja.
type ligacao struct {
	Endereco string // host:porta
	Chave    string
	Aparelho string // identificador deste aparelho (gerado uma vez)
}

var (
	muLigacao   sync.Mutex
	ligacaoAtiv *ligacao
	caminhoLig  string
	clienteHTTP = &http.Client{Timeout: 15 * time.Second}
)

func carregarLigacao(dirDados string) error {
	muLigacao.Lock()
	defer muLigacao.Unlock()
	caminhoLig = filepath.Join(dirDados, "ligacao-balcao.json")
	b, err := os.ReadFile(caminhoLig)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var l ligacao
	if err := json.Unmarshal(b, &l); err != nil {
		return err
	}
	if l.Endereco != "" {
		ligacaoAtiv = &l
	}
	return nil
}

func ligacaoAtual() *ligacao {
	muLigacao.Lock()
	defer muLigacao.Unlock()
	return ligacaoAtiv
}

func gravarLigacao(l *ligacao) error {
	muLigacao.Lock()
	defer muLigacao.Unlock()
	if l == nil {
		ligacaoAtiv = nil
		err := os.Remove(caminhoLig)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	b, err := json.Marshal(l)
	if err != nil {
		return err
	}
	if err := os.WriteFile(caminhoLig, b, 0o600); err != nil {
		return err
	}
	ligacaoAtiv = l
	return nil
}

// normalizarEndereco aceita "192.168.1.10", "192.168.1.10:8765" ou com http://.
func normalizarEndereco(e string) (string, error) {
	e = strings.TrimSpace(e)
	e = strings.TrimPrefix(strings.TrimPrefix(e, "http://"), "https://")
	e = strings.TrimRight(e, "/")
	if e == "" {
		return "", errors.New("escreva o endereço do computador do balcão (ex: 192.168.1.10)")
	}
	if _, _, err := net.SplitHostPort(e); err != nil {
		e = net.JoinHostPort(e, portaServidorPadrao)
	}
	return e, nil
}

type respostaServidor struct {
	OK    bool            `json:"ok"`
	Dados json.RawMessage `json:"dados"`
	Erro  string          `json:"erro"`
}

func pedirServidor(l *ligacao, caminho string, corpo []byte) (json.RawMessage, error) {
	metodo := http.MethodGet
	if corpo != nil {
		metodo = http.MethodPost
	}
	req, err := http.NewRequest(metodo, "http://"+l.Endereco+caminho, bytes.NewReader(corpo))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Chave", l.Chave)
	req.Header.Set("X-Aparelho", l.Aparelho)
	req.Header.Set("Content-Type", "application/json")
	res, err := clienteHTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sem ligação ao computador do balcão (%s): verifique o Wi-Fi e se o PDV está aberto", l.Endereco)
	}
	defer res.Body.Close()
	var r respostaServidor
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("resposta inválida do computador do balcão (%s)", res.Status)
	}
	if !r.OK {
		return nil, errors.New(r.Erro)
	}
	return r.Dados, nil
}

func chamarServidor(l *ligacao, metodo, argsJSON string) (string, error) {
	if argsJSON == "" {
		argsJSON = "[]"
	}
	corpo, err := json.Marshal(struct {
		Metodo string          `json:"metodo"`
		Args   json.RawMessage `json:"args"`
	}{metodo, json.RawMessage(argsJSON)})
	if err != nil {
		return "", err
	}
	dados, err := pedirServidor(l, "/chamar", corpo)
	if err != nil {
		return "", err
	}
	if len(dados) == 0 {
		return "null", nil
	}
	return string(dados), nil
}

// ponte: chamadas da app que não vão para a loja (window.go.ponte.Ponte.X).
type ponte struct{}

type EstadoLigacao struct {
	// ModoGarcom: o aparelho usa o computador do balcão.
	ModoGarcom bool
	Endereco   string
	Aparelho   string
	// TemImpressora: a app nativa sabe imprimir (Sunmi).
	TemImpressora bool
}

func (ponte) EstadoLigacao() EstadoLigacao {
	e := EstadoLigacao{TemImpressora: plataf != nil}
	if l := ligacaoAtual(); l != nil {
		e.ModoGarcom, e.Endereco, e.Aparelho = true, l.Endereco, l.Aparelho
	}
	return e
}

// LigarAoBalcao testa o endereço e a chave e, se responderem, passa o
// aparelho para o modo garçom.
func (p ponte) LigarAoBalcao(endereco, chave string) (EstadoLigacao, error) {
	end, err := normalizarEndereco(endereco)
	if err != nil {
		return EstadoLigacao{}, err
	}
	chave = strings.ToUpper(strings.TrimSpace(chave))
	if chave == "" {
		return EstadoLigacao{}, errors.New("escreva a chave da loja (aparece no PDV do balcão, em Aparelhos)")
	}
	id := ""
	if l := ligacaoAtual(); l != nil {
		id = l.Aparelho
	}
	if id == "" {
		b := make([]byte, 4)
		_, _ = rand.Read(b)
		id = "aparelho-" + hex.EncodeToString(b)
	}
	l := &ligacao{Endereco: end, Chave: chave, Aparelho: id}
	if _, err := pedirServidor(l, "/estado", nil); err != nil {
		return EstadoLigacao{}, err
	}
	if err := gravarLigacao(l); err != nil {
		return EstadoLigacao{}, err
	}
	return p.EstadoLigacao(), nil
}

// DesligarDoBalcao volta a usar a base deste aparelho.
func (p ponte) DesligarDoBalcao() (EstadoLigacao, error) {
	if err := gravarLigacao(nil); err != nil {
		return EstadoLigacao{}, err
	}
	return p.EstadoLigacao(), nil
}

// ImprimirNoAparelho imprime um talão (já em texto, 32 colunas) na
// impressora embutida.
func (ponte) ImprimirNoAparelho(texto string) error {
	if plataf == nil {
		return errors.New("este aparelho não tem impressora")
	}
	return plataf.ImprimirTexto(texto)
}
