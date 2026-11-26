package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

// RestauranteRepository guarda o que é próprio do restaurante: mesas,
// envios para preparação, pedidos impressos, impressoras e sessões de caixa.
type RestauranteRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRestauranteRepository(db *sql.DB) *RestauranteRepository {
	return &RestauranteRepository{db: db, q: sqlcgen.New(db)}
}

func (r *RestauranteRepository) WithTx(ctx context.Context, fn func(*RestauranteRepository) error) error {
	return runInTx(ctx, r.db, func(tx *sql.Tx) error {
		return fn(&RestauranteRepository{db: r.db, q: r.q.WithTx(tx)})
	})
}

// --- Mesas -------------------------------------------------------------------

func (r *RestauranteRepository) DefinirContaMesa(ctx context.Context, contaID int64, mesa string, garcomID, pessoas int64) error {
	return r.q.DefinirContaMesa(ctx, sqlcgen.DefinirContaMesaParams{ContaID: contaID, Mesa: mesa, GarcomID: garcomID, Pessoas: pessoas})
}

func (r *RestauranteRepository) DefinirEtiquetaMesa(ctx context.Context, contaID int64, etiqueta string) error {
	return r.q.DefinirEtiquetaMesa(ctx, sqlcgen.DefinirEtiquetaMesaParams{ContaID: contaID, Etiqueta: etiqueta})
}

// ContaMesa devolve a mesa de uma conta; ok=false se a conta não for de mesa.
func (r *RestauranteRepository) ContaMesa(ctx context.Context, contaID int64) (mesa string, garcomID, pessoas int64, ok bool, err error) {
	row, err := r.q.ObterContaMesa(ctx, contaID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, 0, false, nil
	}
	if err != nil {
		return "", 0, 0, false, err
	}
	return row.Mesa, row.GarcomID, row.Pessoas, true, nil
}

func (r *RestauranteRepository) ListarMesasAbertas(ctx context.Context, negocioID int64) ([]domain.MesaAberta, error) {
	rows, err := r.q.ListarMesasAbertas(ctx, negocioID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.MesaAberta, 0, len(rows))
	for _, m := range rows {
		out = append(out, domain.MesaAberta{
			ContaID: m.ContaID, Mesa: m.Mesa, Etiqueta: m.Etiqueta, ClienteID: m.ClienteID, GarcomID: m.GarcomID, GarcomNome: m.GarcomNome, Pessoas: m.Pessoas,
			AbertaEm: parseTimestamp(m.AbertaEm), Total: m.Total, PorEnviar: m.PorEnviar,
		})
	}
	return out, nil
}

func (r *RestauranteRepository) ListarItens(ctx context.Context, contaID int64) ([]domain.ItemMesa, error) {
	rows, err := r.q.ListarItensComEnvio(ctx, contaID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ItemMesa, 0, len(rows))
	for _, i := range rows {
		out = append(out, domain.ItemMesa{
			ID: i.ID, ContaID: i.ContaID, ProdutoID: i.ProdutoID, UnidadeProdutoID: i.UnidadeProdutoID,
			Produto: i.ProdutoNome, Unidade: i.NomeUnidade, CategoriaID: nullInt64ToPtr(i.CategoriaID),
			Quantidade: i.Quantidade, Enviada: i.Enviada, PrecoUnitario: i.PrecoUnitarioAplicado, Desconto: i.Desconto,
			Subtotal: i.Quantidade*i.PrecoUnitarioAplicado - i.Desconto,
		})
	}
	return out, nil
}

func (r *RestauranteRepository) DefinirQuantidadeEnviada(ctx context.Context, itemID int64, quantidade float64) error {
	return r.q.DefinirQuantidadeEnviada(ctx, sqlcgen.DefinirQuantidadeEnviadaParams{ItemID: itemID, Quantidade: quantidade})
}

// --- Pedidos ------------------------------------------------------------------

func (r *RestauranteRepository) CriarPedido(ctx context.Context, p domain.Pedido) (domain.Pedido, error) {
	row, err := r.q.CriarPedido(ctx, sqlcgen.CriarPedidoParams{
		NegocioID: domain.NegocioID, ContaID: p.ContaID, Tipo: string(p.Tipo), Mesa: p.Mesa, GarcomID: p.GarcomID,
		Motivo: ptrToNullString(p.Motivo), AutorizadoPor: ptrToNullInt64(p.AutorizadoPor),
	})
	if err != nil {
		return domain.Pedido{}, err
	}
	for _, l := range p.Linhas {
		if err := r.q.CriarPedidoLinha(ctx, sqlcgen.CriarPedidoLinhaParams{
			PedidoID: row.ID, ProdutoID: l.ProdutoID, Descricao: l.Descricao, Unidade: l.Unidade,
			Quantidade: l.Quantidade, ImpressoraID: ptrToNullInt64(l.ImpressoraID),
		}); err != nil {
			return domain.Pedido{}, err
		}
	}
	criado := pedidoFromRow(row)
	criado.Linhas = p.Linhas
	return criado, nil
}

func (r *RestauranteRepository) ObterPedido(ctx context.Context, id int64) (domain.Pedido, error) {
	row, err := r.q.ObterPedido(ctx, id)
	if err != nil {
		return domain.Pedido{}, err
	}
	p := pedidoFromRow(row)
	linhas, err := r.q.ListarPedidoLinhas(ctx, id)
	if err != nil {
		return domain.Pedido{}, err
	}
	for _, l := range linhas {
		p.Linhas = append(p.Linhas, domain.LinhaPedido{
			ProdutoID: l.ProdutoID, Descricao: l.Descricao, Unidade: l.Unidade, Quantidade: l.Quantidade,
			ImpressoraID: nullInt64ToPtr(l.ImpressoraID),
		})
	}
	return p, nil
}

func (r *RestauranteRepository) ListarPedidosDaConta(ctx context.Context, contaID int64) ([]domain.Pedido, error) {
	rows, err := r.q.ListarPedidosDaConta(ctx, contaID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Pedido, 0, len(rows))
	for _, row := range rows {
		out = append(out, pedidoFromRow(row))
	}
	return out, nil
}

func pedidoFromRow(row sqlcgen.Pedidos) domain.Pedido {
	return domain.Pedido{
		ID: row.ID, ContaID: row.ContaID, Tipo: domain.TipoPedido(row.Tipo), Mesa: row.Mesa, GarcomID: row.GarcomID,
		Motivo: nullStringToPtr(row.Motivo), AutorizadoPor: nullInt64ToPtr(row.AutorizadoPor), CriadoEm: parseTimestamp(row.CriadoEm),
	}
}

// --- Impressoras ----------------------------------------------------------------

func (r *RestauranteRepository) ListarImpressoras(ctx context.Context) ([]domain.Impressora, error) {
	rows, err := r.q.ListarImpressoras(ctx, domain.NegocioID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Impressora, 0, len(rows))
	for _, row := range rows {
		out = append(out, impressoraFromRow(row))
	}
	return out, nil
}

func (r *RestauranteRepository) ObterImpressora(ctx context.Context, id int64) (domain.Impressora, error) {
	row, err := r.q.ObterImpressora(ctx, id)
	if err != nil {
		return domain.Impressora{}, err
	}
	return impressoraFromRow(row), nil
}

func (r *RestauranteRepository) CriarImpressora(ctx context.Context, i domain.Impressora) (domain.Impressora, error) {
	row, err := r.q.CriarImpressora(ctx, sqlcgen.CriarImpressoraParams{
		NegocioID: domain.NegocioID, Nome: i.Nome, Endereco: i.Endereco, Porta: i.Porta, Colunas: i.Colunas, Funcao: string(i.Funcao),
	})
	if err != nil {
		return domain.Impressora{}, err
	}
	return impressoraFromRow(row), nil
}

func (r *RestauranteRepository) AtualizarImpressora(ctx context.Context, i domain.Impressora) error {
	return r.q.AtualizarImpressora(ctx, sqlcgen.AtualizarImpressoraParams{
		Nome: i.Nome, Endereco: i.Endereco, Porta: i.Porta, Colunas: i.Colunas, Funcao: string(i.Funcao), Ativa: boolToInt64(i.Ativa), ID: i.ID,
	})
}

// ImpressoraPorCategoria: categoria → impressora de preparação.
func (r *RestauranteRepository) ImpressoraPorCategoria(ctx context.Context) (map[int64]int64, error) {
	rows, err := r.q.ListarCategoriaImpressora(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[int64]int64, len(rows))
	for _, row := range rows {
		m[row.CategoriaID] = row.ImpressoraID
	}
	return m, nil
}

func (r *RestauranteRepository) DefinirImpressoraDaCategoria(ctx context.Context, categoriaID int64, impressoraID *int64) error {
	if impressoraID == nil {
		return r.q.RemoverCategoriaImpressora(ctx, categoriaID)
	}
	return r.q.DefinirCategoriaImpressora(ctx, sqlcgen.DefinirCategoriaImpressoraParams{CategoriaID: categoriaID, ImpressoraID: *impressoraID})
}

func impressoraFromRow(row sqlcgen.Impressoras) domain.Impressora {
	return domain.Impressora{
		ID: row.ID, Nome: row.Nome, Endereco: row.Endereco, Porta: row.Porta, Colunas: row.Colunas,
		Funcao: domain.FuncaoImpressora(row.Funcao), Ativa: int64ToBool(row.Ativa),
	}
}

// --- Caixa ---------------------------------------------------------------------

// SessaoAberta devolve a sessão de caixa aberta; ok=false se não houver.
func (r *RestauranteRepository) SessaoAberta(ctx context.Context) (domain.SessaoCaixa, bool, error) {
	row, err := r.q.ObterSessaoCaixaAberta(ctx, domain.NegocioID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SessaoCaixa{}, false, nil
	}
	if err != nil {
		return domain.SessaoCaixa{}, false, err
	}
	return sessaoFromRow(row), true, nil
}

func (r *RestauranteRepository) ObterSessao(ctx context.Context, id int64) (domain.SessaoCaixa, error) {
	row, err := r.q.ObterSessaoCaixa(ctx, id)
	if err != nil {
		return domain.SessaoCaixa{}, err
	}
	return sessaoFromRow(row), nil
}

func (r *RestauranteRepository) AbrirSessao(ctx context.Context, utilizadorID int64, fundo float64) (domain.SessaoCaixa, error) {
	row, err := r.q.AbrirSessaoCaixa(ctx, sqlcgen.AbrirSessaoCaixaParams{NegocioID: domain.NegocioID, UtilizadorAberturaID: utilizadorID, FundoInicial: fundo})
	if err != nil {
		return domain.SessaoCaixa{}, err
	}
	return sessaoFromRow(row), nil
}

func (r *RestauranteRepository) FecharSessao(ctx context.Context, id, utilizadorID int64, contagem map[domain.MetodoPagamento]float64, obs *string) (domain.SessaoCaixa, error) {
	j, err := json.Marshal(contagem)
	if err != nil {
		return domain.SessaoCaixa{}, err
	}
	texto := string(j)
	row, err := r.q.FecharSessaoCaixa(ctx, sqlcgen.FecharSessaoCaixaParams{
		UtilizadorFechoID: sql.NullInt64{Int64: utilizadorID, Valid: true}, Contagem: ptrToNullString(&texto), Observacoes: ptrToNullString(obs), ID: id,
	})
	if err != nil {
		return domain.SessaoCaixa{}, err
	}
	return sessaoFromRow(row), nil
}

func (r *RestauranteRepository) ListarSessoes(ctx context.Context, limite int64) ([]domain.SessaoCaixa, error) {
	rows, err := r.q.ListarSessoesCaixa(ctx, sqlcgen.ListarSessoesCaixaParams{NegocioID: domain.NegocioID, Limit: limite})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SessaoCaixa, 0, len(rows))
	for _, row := range rows {
		out = append(out, sessaoFromRow(row))
	}
	return out, nil
}

func sessaoFromRow(row sqlcgen.SessoesCaixa) domain.SessaoCaixa {
	s := domain.SessaoCaixa{
		ID: row.ID, AbertaEm: parseTimestamp(row.AbertaEm), UtilizadorAberturaID: row.UtilizadorAberturaID,
		FundoInicial: row.FundoInicial, UtilizadorFechoID: nullInt64ToPtr(row.UtilizadorFechoID), Observacoes: nullStringToPtr(row.Observacoes),
	}
	if row.FechadaEm.Valid {
		t := parseTimestamp(row.FechadaEm.String)
		s.FechadaEm = &t
	}
	if row.Contagem.Valid {
		_ = json.Unmarshal([]byte(row.Contagem.String), &s.Contagem)
	}
	return s
}

// WithTxVendas corre fn numa só transação para as mesas e as contas —
// dividir ou juntar mesas mexe nas duas coisas ao mesmo tempo.
func (r *RestauranteRepository) WithTxVendas(ctx context.Context, vendas *VendaRepository, fn func(*RestauranteRepository, *VendaRepository) error) error {
	return runInTx(ctx, r.db, func(tx *sql.Tx) error {
		txStock := &StockRepository{db: r.db, q: sqlcgen.New(tx)}
		txVendas := &VendaRepository{db: vendas.db, q: vendas.q.WithTx(tx), Stock: txStock,
			Lotes: &LoteRepository{db: r.db, q: sqlcgen.New(tx), Stock: txStock}}
		return fn(&RestauranteRepository{db: r.db, q: r.q.WithTx(tx)}, txVendas)
	})
}
