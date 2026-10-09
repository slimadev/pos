package admin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/config"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// VendasService serve o ecrã "Vendas" do cmd/admin: histórico por período,
// detalhe de uma venda e reimpressão do recibo. Os nomes (cliente, produto,
// unidade) são juntados aqui, numa só ligação à loja, para o ecrã não ter
// de fazer uma chamada por produto.
type VendasService struct {
	lojas *repository.LojaRepository
}

func NewVendasService(lojas *repository.LojaRepository) *VendasService {
	return &VendasService{lojas: lojas}
}

type VendaResumo struct {
	Venda        domain.Venda
	ClienteNome  string
	OperadorNome string
}

type ItemVendaDetalhe struct {
	Item        domain.ItemVenda
	ProdutoNome string
	UnidadeNome string
}

type VendaDetalhe struct {
	Venda        domain.Venda
	ClienteNome  string
	OperadorNome string
	Itens        []ItemVendaDetalhe
	Pagamentos   []domain.Pagamento
}

// Listar recebe datas "AAAA-MM-DD", ambas inclusive.
func (s *VendasService) Listar(ctx context.Context, desde, ate string) ([]VendaResumo, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	vendas, err := app.ListarVendasEntreDatas(desde, ate)
	if err != nil {
		return nil, err
	}
	nomeCliente, err := nomesDeClientes(app)
	if err != nil {
		return nil, err
	}
	nomeOperador, err := nomesDeOperadores(ctx, app)
	if err != nil {
		return nil, err
	}

	out := make([]VendaResumo, 0, len(vendas))
	for _, v := range vendas {
		out = append(out, VendaResumo{Venda: v, ClienteNome: nomeCliente(v.ClienteID), OperadorNome: nomeOperador(v.UtilizadorID)})
	}
	return out, nil
}

func (s *VendasService) ObterDetalhe(ctx context.Context, vendaID int64) (VendaDetalhe, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return VendaDetalhe{}, err
	}
	defer fechar()

	completa, err := app.ObterVendaCompleta(vendaID)
	if err != nil {
		return VendaDetalhe{}, err
	}
	nomeCliente, err := nomesDeClientes(app)
	if err != nil {
		return VendaDetalhe{}, err
	}
	nomeOperador, err := nomesDeOperadores(ctx, app)
	if err != nil {
		return VendaDetalhe{}, err
	}
	produtos, err := app.ListarProdutos()
	if err != nil {
		return VendaDetalhe{}, err
	}
	nomeProduto := make(map[int64]string, len(produtos))
	for _, p := range produtos {
		nomeProduto[p.ID] = p.Nome
	}

	nomeUnidade := make(map[int64]string)
	unidadesCarregadas := make(map[int64]bool)
	itens := make([]ItemVendaDetalhe, 0, len(completa.Itens))
	for _, item := range completa.Itens {
		if !unidadesCarregadas[item.ProdutoID] {
			unidades, err := app.ListarUnidadesProduto(item.ProdutoID)
			if err != nil {
				return VendaDetalhe{}, err
			}
			for _, u := range unidades {
				nomeUnidade[u.ID] = u.NomeUnidade
			}
			unidadesCarregadas[item.ProdutoID] = true
		}
		produtoNome, ok := nomeProduto[item.ProdutoID]
		if !ok {
			produtoNome = fmt.Sprintf("Produto #%d", item.ProdutoID)
		}
		itens = append(itens, ItemVendaDetalhe{Item: item, ProdutoNome: produtoNome, UnidadeNome: nomeUnidade[item.UnidadeProdutoID]})
	}

	return VendaDetalhe{
		Venda:        completa.Venda,
		ClienteNome:  nomeCliente(completa.Venda.ClienteID),
		OperadorNome: nomeOperador(completa.Venda.UtilizadorID),
		Itens:        itens,
		Pagamentos:   completa.Pagamentos,
	}, nil
}

// GerarRecibo grava o PDF e devolve o caminho. Não o abre: quem abre é o
// AdminApp, com o contexto Wails do cmd/admin — o api.App aberto por
// AbrirLoja não tem um, e GerarRecibo do cmd/app usaria esse contexto.
func (s *VendasService) GerarRecibo(ctx context.Context, vendaID int64) (string, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return "", err
	}
	defer fechar()

	pdf, err := app.ReciboService.GerarPDF(ctx, vendaID)
	if err != nil {
		return "", err
	}
	dir, err := config.ResolveRecibosDir()
	if err != nil {
		return "", err
	}
	caminho := filepath.Join(dir, fmt.Sprintf("recibo-%d.pdf", vendaID))
	if err := os.WriteFile(caminho, pdf, 0o644); err != nil {
		return "", fmt.Errorf("gravar recibo: %w", err)
	}
	return caminho, nil
}

// nomesDeOperadores: as vendas sem operador identificado (lojas sem
// operadores, ou feitas no admin) ficam com o administrador.
func nomesDeOperadores(ctx context.Context, app *api.App) (func(int64) string, error) {
	operadores, err := app.OperadorService.Listar(ctx)
	if err != nil {
		return nil, err
	}
	nomes := make(map[int64]string, len(operadores))
	for _, o := range operadores {
		nomes[o.ID] = o.Nome
	}
	return func(id int64) string {
		if n, ok := nomes[id]; ok {
			return n
		}
		if id == api.UtilizadorAdminID {
			return "Administrador"
		}
		return fmt.Sprintf("Utilizador #%d", id)
	}, nil
}

func nomesDeClientes(app *api.App) (func(int64) string, error) {
	clientes, err := app.ListarClientes()
	if err != nil {
		return nil, err
	}
	nomes := make(map[int64]string, len(clientes))
	for _, c := range clientes {
		nomes[c.ID] = c.Nome
	}
	return func(id int64) string {
		if n, ok := nomes[id]; ok {
			return n
		}
		if id == domain.ClienteAnonimoID {
			return "Cliente anónimo"
		}
		return fmt.Sprintf("Cliente #%d", id)
	}, nil
}

func (s *VendasService) abrirLoja(ctx context.Context) (*api.App, func(), error) {
	return abrirLojaUnica(ctx, s.lojas)
}
