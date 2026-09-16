-- Fase 1 do plano de implementação — core do SDD (secção 3.2): negócio,
-- armazéns, produtos, unidades e o motor de movimentos de stock.
--
-- `utilizadores` entra já aqui, não só na Fase 2, porque todo movimento de
-- stock exige utilizador_id para auditoria (SDD secção 10) desde o
-- primeiro registo — a tabela tem de existir antes da FK poder existir.

CREATE TABLE negocios (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nome TEXT NOT NULL,
    tipo_negocio TEXT NOT NULL CHECK (tipo_negocio IN ('mercearia', 'ferragem', 'farmacia', 'bottlestore', 'personalizado')),
    ativo INTEGER NOT NULL DEFAULT 1
);

-- Cada executável cmd/app serve sempre uma única loja (SDD secção 2.1);
-- id=1 é essa loja. O wizard da Fase 2 atualiza esta linha em vez de criar
-- outra — nunca deve existir um segundo negócio na base de uma loja.
INSERT INTO negocios (id, nome, tipo_negocio, ativo) VALUES (1, 'Loja por configurar', 'personalizado', 1);

CREATE TABLE configuracoes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER NOT NULL REFERENCES negocios (id),
    chave TEXT NOT NULL,
    valor TEXT NOT NULL,
    UNIQUE (negocio_id, chave)
);

CREATE TABLE utilizadores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER REFERENCES negocios (id),
    nome TEXT NOT NULL,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'gerente', 'operador_pos')),
    ativo INTEGER NOT NULL DEFAULT 1
);

-- Utilizador de arranque para os movimentos terem um utilizador_id válido
-- antes do wizard (SDD secção 9.2, passo 4) criar o admin real com login.
-- password_hash vazio = ainda não há autenticação aplicada (Fase 2).
INSERT INTO utilizadores (id, negocio_id, nome, username, password_hash, role, ativo)
VALUES (1, 1, 'Administrador', 'admin', '', 'admin', 1);

CREATE TABLE armazens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER NOT NULL REFERENCES negocios (id),
    nome TEXT NOT NULL,
    tipo TEXT NOT NULL CHECK (tipo IN ('principal', 'ponto_venda', 'deposito')),
    ativo INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE categorias (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER NOT NULL REFERENCES negocios (id),
    nome TEXT NOT NULL,
    categoria_pai_id INTEGER REFERENCES categorias (id)
);

CREATE TABLE produtos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER NOT NULL REFERENCES negocios (id),
    categoria_id INTEGER REFERENCES categorias (id),
    nome TEXT NOT NULL,
    tipo_produto TEXT NOT NULL CHECK (tipo_produto IN ('simples', 'composto', 'semi_elaborado')),
    unidade_base TEXT NOT NULL,
    stock_minimo REAL,
    -- SDD 3.2: só simples/semi_elaborado controlam stock; composto é
    -- sempre calculado via ficha técnica (Fase 5), nunca guardado aqui.
    controla_stock INTEGER NOT NULL DEFAULT 1,
    ativo INTEGER NOT NULL DEFAULT 1,
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    atualizado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX idx_produtos_negocio ON produtos (negocio_id);

CREATE TABLE unidades_produto (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    produto_id INTEGER NOT NULL REFERENCES produtos (id),
    nome_unidade TEXT NOT NULL,
    qtd_em_unidade_base REAL NOT NULL CHECK (qtd_em_unidade_base > 0),
    preco_custo REAL NOT NULL DEFAULT 0,
    preco_venda REAL NOT NULL DEFAULT 0,
    codigo_barras TEXT,
    is_padrao_venda INTEGER NOT NULL DEFAULT 0,
    is_padrao_compra INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_unidades_produto_produto ON unidades_produto (produto_id);
CREATE UNIQUE INDEX idx_unidades_produto_codigo_barras ON unidades_produto (codigo_barras) WHERE codigo_barras IS NOT NULL;

CREATE TABLE movimentos_stock (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    produto_id INTEGER NOT NULL REFERENCES produtos (id),
    armazem_origem_id INTEGER REFERENCES armazens (id),
    armazem_destino_id INTEGER REFERENCES armazens (id),
    -- venda/producao entram no schema já agora (mesmo schema para todas as
    -- fases, SDD 8.4) mas só passam a ser emitidos a partir da Fase 3/5.
    tipo TEXT NOT NULL CHECK (tipo IN ('entrada', 'transferencia', 'perda', 'consumo_interno', 'ajuste_inventario', 'venda', 'producao')),
    quantidade_unidade_base REAL NOT NULL CHECK (quantidade_unidade_base > 0),
    custo_unitario REAL,
    unidade_produto_id INTEGER REFERENCES unidades_produto (id),
    motivo TEXT,
    referencia_id TEXT,
    utilizador_id INTEGER NOT NULL REFERENCES utilizadores (id),
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK (armazem_origem_id IS NOT NULL OR armazem_destino_id IS NOT NULL)
);

CREATE INDEX idx_movimentos_produto ON movimentos_stock (produto_id);
CREATE INDEX idx_movimentos_criado_em ON movimentos_stock (criado_em);

-- Custo médio ponderado por (produto, armazém), como a própria stock_atual
-- do SDD 3.2 pareia as duas colunas. Mantida por transação em
-- StockService em vez de recalculada por view (SDD 3.2 aceita as duas
-- alternativas; uma tabela evita reimplementar média ponderada sequencial
-- dentro de uma view SQL).
CREATE TABLE produto_custo_medio (
    produto_id INTEGER NOT NULL REFERENCES produtos (id),
    armazem_id INTEGER NOT NULL REFERENCES armazens (id),
    custo_medio_ponderado REAL NOT NULL DEFAULT 0,
    PRIMARY KEY (produto_id, armazem_id)
);

-- stock_atual como view (recomendação do SDD 3.2): nunca pode divergir dos
-- movimentos porque é sempre recalculada a partir deles. Produtos
-- `composto` nunca aparecem aqui — nunca geram movimentos por si próprios
-- (só os seus ingredientes geram, a partir da Fase 5).
CREATE VIEW stock_atual AS
SELECT
    m.produto_id AS produto_id,
    m.armazem_id AS armazem_id,
    SUM(m.delta) AS quantidade_unidade_base,
    COALESCE(pcm.custo_medio_ponderado, 0) AS custo_medio_ponderado
FROM (
    SELECT produto_id, armazem_destino_id AS armazem_id, quantidade_unidade_base AS delta
    FROM movimentos_stock WHERE armazem_destino_id IS NOT NULL
    UNION ALL
    SELECT produto_id, armazem_origem_id AS armazem_id, -quantidade_unidade_base AS delta
    FROM movimentos_stock WHERE armazem_origem_id IS NOT NULL
) m
LEFT JOIN produto_custo_medio pcm ON pcm.produto_id = m.produto_id AND pcm.armazem_id = m.armazem_id
GROUP BY m.produto_id, m.armazem_id;

CREATE TABLE inventarios (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER NOT NULL REFERENCES negocios (id),
    armazem_id INTEGER NOT NULL REFERENCES armazens (id),
    data TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    status TEXT NOT NULL CHECK (status IN ('aberto', 'fechado')) DEFAULT 'aberto',
    utilizador_id INTEGER NOT NULL REFERENCES utilizadores (id)
);

CREATE TABLE inventario_itens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    inventario_id INTEGER NOT NULL REFERENCES inventarios (id),
    produto_id INTEGER NOT NULL REFERENCES produtos (id),
    quantidade_sistema REAL NOT NULL,
    quantidade_contada REAL,
    diferenca REAL,
    ajustado INTEGER NOT NULL DEFAULT 0,
    UNIQUE (inventario_id, produto_id)
);

CREATE INDEX idx_inventario_itens_inventario ON inventario_itens (inventario_id);
