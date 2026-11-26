// Package servidor é o que o PC do balcão abre na rede local para os
// aparelhos dos garçons (Sunmi V2 Pro…): cada aparelho chama os métodos do
// api.App por HTTP, como a app chamaria o backend local, e o PC responde a
// partir da única base de dados da loja. Assim as mesas, os pedidos e o
// caixa são os mesmos em todos os aparelhos.
//
// Protocolo: POST /chamar com {"metodo": "...", "args": [...]} e os
// cabeçalhos X-Chave (a chave da loja) e X-Aparelho (identificador do
// aparelho). Resposta {"ok": true, "dados": ...} ou {"ok": false, "erro": "..."}.
package servidor

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/despacho"
)

// metodosPermitidos: o que um aparelho de garçom pode fazer. O resto
// (produtos, preços, fechar contas, caixa…) fica no PC.
var metodosPermitidos = map[string]bool{
	"Ping": true, "EstaConfigurado": true, "ObterEstadoConfig": true,
	"ListarOperadoresAtivos": true, "IniciarTurno": true, "TerminarTurno": true, "ObterOperadorAtual": true,
	"CatalogoPDV": true, "ListarArmazens": true, "ListarCategorias": true,
	"ListarMesas": true, "ObterMesa": true, "AbrirMesa": true, "LancarItemMesa": true, "DefinirPessoasMesa": true,
	"EnviarPedido": true, "ListarPedidosMesa": true, "ReimprimirPedido": true, "AnularItemMesa": true,
	"DividirMesa": true, "TransferirMesa": true, "JuntarMesas": true, "ImprimirContaMesa": true,
	"ListarImpressoras": true, "AutorizarGerente": true,
}

type aparelho struct {
	app   *api.App
	visto time.Time
}

type Servidor struct {
	base  *api.App
	chave func() string

	mu        sync.Mutex
	aparelhos map[string]*aparelho
	agora     func() time.Time
}

// Novo cria o servidor sobre o api.App do PDV. chave devolve a chave atual
// (lida a cada pedido, para uma chave nova valer logo).
func Novo(base *api.App, chave func() string) *Servidor {
	return &Servidor{base: base, chave: chave, aparelhos: map[string]*aparelho{}, agora: time.Now}
}

type pedido struct {
	Metodo string          `json:"metodo"`
	Args   json.RawMessage `json:"args"`
}

type resposta struct {
	OK    bool            `json:"ok"`
	Dados json.RawMessage `json:"dados,omitempty"`
	Erro  string          `json:"erro,omitempty"`
}

func (s *Servidor) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/chamar", s.chamar)
	mux.HandleFunc("/estado", s.estado)
	return mux
}

func escrever(w http.ResponseWriter, codigo int, r resposta) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	_ = json.NewEncoder(w).Encode(r)
}

func (s *Servidor) autorizado(r *http.Request) bool {
	esperada := s.chave()
	dada := r.Header.Get("X-Chave")
	return esperada != "" && subtle.ConstantTimeCompare([]byte(esperada), []byte(dada)) == 1
}

// estado serve para o aparelho testar a ligação e a chave.
func (s *Servidor) estado(w http.ResponseWriter, r *http.Request) {
	if !s.autorizado(r) {
		escrever(w, http.StatusUnauthorized, resposta{Erro: "chave da loja errada"})
		return
	}
	dados, err := despacho.Chamar(s.base, "ObterEstadoConfig", "")
	if err != nil {
		escrever(w, http.StatusOK, resposta{Erro: err.Error()})
		return
	}
	escrever(w, http.StatusOK, resposta{OK: true, Dados: json.RawMessage(dados)})
}

func (s *Servidor) chamar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		escrever(w, http.StatusMethodNotAllowed, resposta{Erro: "use POST"})
		return
	}
	if !s.autorizado(r) {
		escrever(w, http.StatusUnauthorized, resposta{Erro: "chave da loja errada"})
		return
	}
	id := r.Header.Get("X-Aparelho")
	if id == "" || len(id) > 100 {
		escrever(w, http.StatusBadRequest, resposta{Erro: "falta o identificador do aparelho"})
		return
	}
	var p pedido
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&p); err != nil {
		escrever(w, http.StatusBadRequest, resposta{Erro: "pedido inválido"})
		return
	}
	if !metodosPermitidos[p.Metodo] {
		escrever(w, http.StatusOK, resposta{Erro: "esta operação só se faz no computador do balcão"})
		return
	}
	dados, err := despacho.Chamar(s.app(id), p.Metodo, string(p.Args))
	if err != nil {
		escrever(w, http.StatusOK, resposta{Erro: err.Error()})
		return
	}
	escrever(w, http.StatusOK, resposta{OK: true, Dados: json.RawMessage(dados)})
}

// app devolve o api.App do aparelho (com o seu turno), criando-o à primeira.
func (s *Servidor) app(id string) *api.App {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.aparelhos[id]
	if !ok {
		a = &aparelho{app: s.base.ParaDispositivo()}
		s.aparelhos[id] = a
	}
	a.visto = s.agora()
	return a.app
}

func (s *Servidor) Aparelhos() []api.AparelhoLigado {
	s.mu.Lock()
	lista := make(map[string]aparelho, len(s.aparelhos))
	for id, a := range s.aparelhos {
		lista[id] = *a
	}
	s.mu.Unlock()
	out := make([]api.AparelhoLigado, 0, len(lista))
	for id, a := range lista {
		op, _ := a.app.ObterOperadorAtual()
		out = append(out, api.AparelhoLigado{ID: id, Operador: op.Nome, Visto: a.visto})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Iniciar abre a porta e atende em segundo plano até parar() ser chamado.
func (s *Servidor) Iniciar(porta int) (parar func(), err error) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(porta))
	if err != nil {
		return nil, err
	}
	srv := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}, nil
}

// EnderecosLocais devolve os IPv4 deste PC na rede local — o que se
// escreve no aparelho do garçom.
func EnderecosLocais() []string {
	var out []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, i := range ifaces {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil {
				out = append(out, ipn.IP.String())
			}
		}
	}
	return out
}
