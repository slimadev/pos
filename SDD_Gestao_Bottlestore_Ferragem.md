# SDD — Sistema de Gestão de Retalho Multi-Negócio
### Software Design Document | v2.0

**Stack:** Go (backend) + Uber Fx (injeção de dependências) + TypeScript/React (frontend) + Wails v2 (bridge desktop, bindings nativos) + SQLite (WAL mode)
**Modelo:** Uma única aplicação genérica, configurável no primeiro arranque para operar como Mercearia, Ferragem, Farmácia ou Bottlestore (bebidas + comida) — ou uma combinação personalizada de módulos —, mais um módulo de Administração universal capaz de agregar várias lojas, mesmo de tipos diferentes.

---

## 1. Introdução e Objetivo

O sistema tem como objetivo gerir negócios de retalho de tipos diferentes (mercearia, ferragem, farmácia, bottlestore com venda de comida) com **um único motor de dados, regras de negócio e instalador**, adaptando-se ao tipo de negócio através de configuração escolhida pelo utilizador no primeiro arranque — sem necessidade de builds separados por negócio.

Exemplos de fluxos que o sistema cobre nativamente:
- **Ferragem / Mercearia / Farmácia**: ponto de venda direto — o cliente compra, paga, sai.
- **Bottlestore**: venda com "contas abertas" (estilo mesa de restaurante, sem mesas físicas), consumo lançado ao longo do tempo, fecho de conta com pagamento ou dívida.
- **Farmácia**: controlo de lotes/validade e receitas médicas.
- **Restauração** (bottlestore com comida, ou outro negócio com pratos preparados): produtos compostos com ficha técnica (receita), consumindo múltiplos ingredientes por venda.

Todos os negócios partilham o mesmo core: gestão de produtos, armazéns/stock, clientes, movimentos de stock, relatórios e exportações (Excel/PDF). O que muda entre tipos de negócio são **módulos opcionais** ativados por configuração (ver secção 9).

Cada loja corre como aplicação desktop standalone (o mesmo executável, instalado em cada loja), com base de dados local SQLite e configuração própria definida no wizard de primeiro arranque. Um módulo de Administração (que pode correr localmente em cada loja, ou centralizado, ver secção 8.5) permite gerir tudo com visão universal, mesmo entre lojas de tipos diferentes.

---

## 2. Visão Geral da Arquitetura

```
┌─────────────────────────────────────────────────────────┐
│              Aplicação Desktop Genérica (Wails)            │
│                                                             │
│   Frontend React/TS          Go Backend (embutido)         │
│   ┌──────────────┐           ┌───────────────────────┐    │
│   │ Menu dinâmico  │◄────────►│  App Layer (usecases)  │    │
│   │ (conforme       │  Wails   │  - ConfigService        │   │
│   │  módulos ativos)│  Bindings │  - ProdutoService       │   │
│   ├──────────────┤           │  - StockService         │    │
│   │ Wizard 1º       │◄────────►│  - ContaService          │   │
│   │ arranque        │           │  - VendaService          │   │
│   ├──────────────┤           │  - ClienteService         │  │
│   │ Ecrãs de venda  │◄────────►│  - FichaTecnicaService    │  │
│   │ (POS/Contas)    │           │  - RelatorioService       │  │
│   │ conforme tipo   │           │  - ExportService (xlsx/pdf)│ │
│   └──────────────┘           └───────────┬───────────────┘ │
│                                            │                │
│                                   ┌────────▼────────┐       │
│                                   │  Repository Layer │      │
│                                   │  (SQL + sqlc/gorm) │     │
│                                   └────────┬────────┘       │
│                                            │                │
│                                   ┌────────▼────────┐       │
│                                   │ SQLite (WAL mode) │      │
│                                   │  + config.json     │     │
│                                   └───────────────────┘      │
└─────────────────────────────────────────────────────────┘
```

### 2.1 Princípios de arquitetura

- **Backend único em Go**, organizado em camadas: `domain` (entidades e regras), `usecase`/`service` (orquestração), `repository` (acesso a dados), `api` (struct `App` com os bindings expostos ao Wails). A montagem destas camadas é feita via **Uber Fx** (ver secção 8.2), garantindo que o executável único e o admin reaproveitam exatamente o mesmo grafo de dependências.
- **Um único frontend React**, cujo menu, ecrãs de venda e tema visual são **renderizados condicionalmente** consoante os módulos ativos e o tipo de negócio configurado (ver secção 9) — não há mais frontends separados por negócio.
- **Um frontend/instância de Administração** (pode correr como app separada ou como área protegida dentro da própria app da loja), capaz de agregar dados de várias lojas, mesmo de tipos diferentes, porque todas partilham o mesmo core de dados.
- **Base de dados**: cada loja tem o seu ficheiro `.db` local mais um `config.json` com o tipo de negócio e módulos ativos. O admin universal lê/escreve diretamente se correr na mesma máquina/rede, ou agrega dados via exportação/sincronização periódica se for centralizado (ver 8.5).
- **Multi-armazém nativo**: cada loja pode ter N armazéns (principal, ponto de venda, depósito, etc.), todos representados da mesma forma — a distinção "principal" ou "ponto de venda" é apenas um atributo (`tipo`), não uma estrutura diferente.

### 2.2 Estrutura de pastas sugerida

```
/backend
  /domain        → entidades e regras puras (Produto, Conta, Venda, FichaTecnica...)
  /usecase       → ConfigService, StockService, ContaService, VendaService, RelatorioService...
  /repository    → sqlite implementations
  /api           → struct App exposto ao Wails (bindings)
  /export        → geração Excel (excelize) e PDF (gofpdf / go-wkhtmltopdf)
/cmd
  /app           → main.go único, genérico — lê config.json, arranca Wails
  /admin         → main.go do admin universal (multi-loja, multi-tipo)
/frontend
  /components    → ProductGrid, Cart, ClientPicker, ReportTable... (partilhados)
  /modules       → ContasAbertas, FichasTecnicas, Lotes, ReceitasMedicas... (renderizados condicionalmente)
  /themes        → mercearia.css, ferragem.css, farmacia.css, bottlestore.css
  /wizard        → ecrã de configuração do primeiro arranque
  /hooks
  /types
/frontend-admin
  /pages         → Produtos, Armazens, Stock, Clientes, Relatorios, Utilizadores, Lojas...
```

---

## 3. Modelo de Dados

### 3.1 Diagrama de entidades (relações principais)

```
armazens ──┬──< movimentos_stock >──┬── produtos ──< unidades_produto
           │                         │
           └──< inventarios          └──< itens_conta / itens_venda

clientes ──< contas ──< itens_conta
clientes ──< vendas
clientes ──< dividas

contas ──1:1── venda (ao fechar, gera uma venda)
vendas ──< itens_venda ──> produtos
vendas ──< pagamentos
dividas ──< pagamentos_divida

utilizadores ──< movimentos_stock (quem registou)
utilizadores ──< vendas (quem vendeu/fechou)
```

### 3.2 Entidades detalhadas

#### `negocios`
| Campo | Tipo | Notas |
|---|---|---|
| id | PK | |
| nome | text | ex: "Mercearia da Dona Ana", "Bottlestore XPTO" |
| tipo_negocio | enum | `mercearia`, `ferragem`, `farmacia`, `bottlestore`, `personalizado` — definido no wizard de 1º arranque (ver secção 9) |
| ativo | bool | |

#### `configuracoes` (módulos ativos e definições da instância)
| Campo | Tipo | Notas |
|---|---|---|
| id | PK | |
| negocio_id | FK | |
| chave | text | ex: `modulo_contas_abertas`, `modulo_fichas_tecnicas`, `modulo_lotes_validade`, `modulo_receitas_medicas`, `modulo_venda_granel`, `modulo_dividas`, `tema` |
| valor | text | ex: `"true"`, `"false"`, `"mercearia"` |

> Gravada no SQLite (consultável por qualquer service em runtime) e espelhada num `config.json` local lido no arranque da app antes de qualquer ligação à base de dados — ver secção 9.3 para o detalhe completo do mecanismo.

#### `armazens`
| Campo | Tipo | Notas |
|---|---|---|
| id | PK | |
| negocio_id | FK | |
| nome | text | ex: "Armazém Principal", "Balcão" |
| tipo | enum | `principal`, `ponto_venda`, `deposito` |
| ativo | bool | |

#### `categorias`
| id, negocio_id, nome, categoria_pai_id (opcional, para subcategorias) |

#### `produtos`
| Campo | Tipo | Notas |
|---|---|---|
| id | PK | |
| negocio_id | FK | |
| categoria_id | FK | |
| nome | text | |
| tipo_produto | enum | `simples` (matéria-prima/revenda), `composto` (prato/bebida com receita), `semi_elaborado` (produzido em lote, usado como ingrediente) |
| unidade_base | text | ex: "unidade", "kg", "litro", "ml" |
| stock_minimo | decimal | alerta de reposição (não aplicável a `composto`) |
| controla_stock | bool | `false` para produtos `composto` — stock é sempre calculado via receita, nunca guardado diretamente |
| ativo | bool | |
| criado_em / atualizado_em | timestamp | |

> **Regra chave:** só produtos `simples` e `semi_elaborado` têm stock real (`controla_stock = true`). Produtos `composto` (ex: "¼ Frango com acompanhamento", "Caipirinha de Vodka") nunca têm stock próprio — o seu "stock disponível" é sempre inferido em tempo real a partir do stock dos ingredientes na respetiva `ficha_tecnica` (ver 3.3).

#### `unidades_produto` (conversões — ver secção anterior sobre caixa/embalagem/unidade, e também doses de bebida: simples/duplo/garrafa)
| id, produto_id, nome_unidade, qtd_em_unidade_base, preco_custo, preco_venda, codigo_barras, is_padrao_venda, is_padrao_compra |
> Cobre tanto "Caixa (24) / Unidade" como "Garrafa (750ml) / Dose Simples (50ml) / Dose Dupla (100ml)" — é o mecanismo para **fracionamento do mesmo produto**. Para produtos que combinam **vários** produtos diferentes (frango + arroz + batata, ou vodka + limão + açúcar), usa-se a tabela `fichas_tecnicas` (3.3), não esta.

#### `fornecedores` (opcional, útil para histórico de compras)
| id, negocio_id, nome, contacto, nif |

#### `stock_atual` (view materializada ou tabela calculada)
| produto_id, armazem_id, quantidade_unidade_base, custo_medio_ponderado |
> Pode ser uma **view SQL** calculada a partir de `movimentos_stock`, evitando inconsistência (recomendado), ou uma tabela mantida por trigger/transação para performance. Só existe para produtos `simples`/`semi_elaborado` — produtos `composto` não aparecem aqui.

#### `movimentos_stock`
| Campo | Tipo | Notas |
|---|---|---|
| id | PK | |
| produto_id | FK | sempre um produto `simples` ou `semi_elaborado` (nunca `composto`) |
| armazem_origem_id | FK nullable | null em entradas/produção |
| armazem_destino_id | FK nullable | null em saídas (perda, consumo, venda) |
| tipo | enum | `entrada`, `transferencia`, `perda`, `consumo_interno`, `ajuste_inventario`, `venda`, `producao` |
| quantidade_unidade_base | decimal | sempre em unidade base |
| custo_unitario | decimal | custo no momento (relevante em entradas/produção) |
| unidade_produto_id | FK | qual unidade foi usada no registo (caixa/unidade/dose/...) para audit |
| motivo | text | obrigatório em perdas/ajustes |
| referencia_id | text/FK | id da venda/conta/inventário/produção relacionado, se aplicável |
| utilizador_id | FK | quem registou |
| criado_em | timestamp | |

> Novo tipo **`producao`**: usado quando um `semi_elaborado` é preparado em lote (ver 3.3.2) — consome os ingredientes da sua receita e cria stock do semi-elaborado.

#### `inventarios` (contagens físicas)
| id, negocio_id, armazem_id, data, status (`aberto`,`fechado`), utilizador_id |
> Aplica-se a produtos `simples` e `semi_elaborado`. É o mecanismo que corrige desvios entre o que a receita previu consumir (ex: 0.30kg de frango por dose) e o que realmente foi usado (perdas de corte, variação de peso da ave, etc.) — ver 3.3.

#### `inventario_itens`
| inventario_id, produto_id, quantidade_sistema, quantidade_contada, diferenca (calculado), ajustado (bool) |

### 3.3 Produtos Compostos e Fichas Técnicas (Receitas)

Cobre pratos de restauração (ex: "¼ Frango com arroz, batata e salada") e bebidas preparadas (ex: "Caipirinha de Vodka") — produtos vendidos diretamente ao cliente, mas cujo stock **não é guardado diretamente**: é calculado a partir do consumo dos ingredientes reais.

#### `fichas_tecnicas` (ficha técnica — não confundir com `receitas_medicas` da farmácia, ver secção 9)
| Campo | Tipo | Notas |
|---|---|---|
| id | PK | |
| produto_composto_id | FK produtos | o prato/bebida vendido (tipo `composto` ou `semi_elaborado`) |
| produto_ingrediente_id | FK produtos | matéria-prima ou semi-elaborado consumido |
| quantidade | decimal | na unidade base do ingrediente (ex: 0.05 L de vodka, 0.30 kg de frango) |
| perda_percentual | decimal | opcional, ex: 10% no descasque de batata — aumenta o consumo real face ao "líquido" da receita |

Um produto `composto` tem **N linhas** em `fichas_tecnicas` (uma por ingrediente).

**Exemplo — Caipirinha de Vodka** (`unidade_base` de cada ingrediente entre parêntesis):
| Ingrediente | Quantidade |
|---|---|
| Vodka (L) | 0.05 |
| Limão (unidade) | 1 |
| Açúcar (g) | 20 |
| Gelo (g) | 100 |

**Exemplo — ¼ Frango com arroz, batata e salada**:
| Ingrediente | Quantidade |
|---|---|
| Frango cru (kg) | 0.30 |
| Arroz cozinhado — semi-elaborado (kg) | 0.15 |
| Batata (kg) | 0.15 |
| Alface/salada (kg) | 0.05 |
| Óleo (L) | 0.02 |

> **Nota sobre peso variável (frango):** a receita usa uma quantidade **média estimada** (ex: 0.30kg), já que o peso real de corte varia de ave para ave. O sistema não tenta pesar cada prato — o **inventário periódico** é o mecanismo que reconcilia a diferença entre o consumo teórico (receita × vendas) e o consumo real, revelando perdas de corte ou desvios de porção. Isto é esperado e normal em restauração; nenhum sistema sem pesagem em cada prato acerta ao grama.

#### 3.3.1 Fluxo de venda de um produto composto

```
Vender 1x "Caipirinha Vodka"
   │
   ├─► Sistema expande a receita do produto
   ├─► Para cada ingrediente: gera movimento_stock tipo `venda`
   │       -0.05 L Vodka
   │       -1 unidade Limão
   │       -20g Açúcar
   │       -100g Gelo
   ├─► Custo do item = Σ (quantidade_receita × custo_médio_atual_do_ingrediente)
   └─► Regista item_venda: "1x Caipirinha Vodka" @ preço_venda, custo = valor calculado acima
```
O cliente vê apenas "1x Caipirinha Vodka" no recibo; por trás, o sistema desce stock de cada ingrediente individualmente. O **lucro real por prato/bebida** fica sempre correto mesmo que o custo do limão ou da vodka mude ao longo do tempo, porque o custo é calculado **no momento da venda** (mesmo princípio já usado em `custo_unitario_no_momento` nos itens de venda simples).

> Se algum ingrediente não tiver stock suficiente no armazém de venda, o sistema deve bloquear ou alertar (configurável) — tal como bloquearia a venda de um produto simples sem stock.

#### 3.3.2 Semi-elaborados e produção em lote

Para itens preparados antecipadamente em quantidade (ex: "Arroz Cozinhado", "Xarope de Caipirinha"):
1. O semi-elaborado é um produto normal com `tipo_produto = semi_elaborado` e `controla_stock = true`, e tem a sua própria `ficha_tecnica` (ex: Arroz Cozinhado = Arroz cru + Água + Sal).
2. Ao **produzir** um lote (ex: cozinhar 5kg de arroz), regista-se um movimento tipo `producao`: consome os ingredientes da receita do semi-elaborado e **cria stock** do semi-elaborado no armazém indicado (ex: cozinha).
3. O prato final (ex: "¼ Frango com arroz") pode então referenciar na sua receita o ingrediente **"Arroz Cozinhado"** (semi-elaborado) em vez de "Arroz cru + Água" — refletindo melhor o fluxo real da cozinha e evitando reexpandir receitas em vários níveis a cada venda.

```
movimento_stock tipo `producao`:
  Produzir 5kg de "Arroz Cozinhado"
     │
     ├─► Consome (via receita do semi-elaborado): -1.2kg Arroz cru, -X L Água, -Y g Sal
     └─► Cria stock: +5kg "Arroz Cozinhado" no armazém "Cozinha"
```

Isto permite **receitas em vários níveis** (prato final usa semi-elaborado, que por sua vez tem a sua própria receita), mantendo cada camada simples de gerir.

### 3.4 Resumo dos mecanismos de fracionamento/composição

| Mecanismo | Para quê | Exemplo |
|---|---|---|
| `unidades_produto` | Fracionamento do **mesmo** produto | Caixa(24)/Unidade de cerveja; Garrafa/Dose Simples/Dose Dupla de whisky |
| `fichas_tecnicas` | Composição de **vários** produtos num item vendido | Caipirinha (vodka+limão+açúcar+gelo); ¼ Frango c/ acompanhamento |
| Movimento `producao` | Preparo em lote de semi-elaborados | Cozinhar arroz, preparar xarope |
| Inventário periódico | Corrige desvios entre receita teórica e consumo real | Perdas de corte do frango, variação de peso |

#### `clientes`
| id, negocio_id, nome, telefone, nif (opcional), morada, limite_credito (opcional), ativo |
> Cliente "anónimo" = registo fixo (`id=0` ou flag `is_anonimo`) usado por omissão.

#### `contas` (específico da bottlestore, mas modelo genérico — ver 3.3)
| Campo | Tipo | Notas |
|---|---|---|
| id | PK | |
| negocio_id | FK | |
| cliente_id | FK nullable | null = anónimo |
| identificador | text | nome livre, ex: "Mesa 3", "Grupo do João", número sequencial |
| status | enum | `aberta`, `fechada`, `cancelada` |
| aberta_em | timestamp | |
| fechada_em | timestamp nullable | |
| utilizador_abertura_id | FK | |
| utilizador_fecho_id | FK nullable | |

#### `itens_conta`
| id, conta_id, produto_id, unidade_produto_id, quantidade, preco_unitario_aplicado, desconto, criado_em, utilizador_id |

#### `vendas`
| Campo | Tipo | Notas |
|---|---|---|
| id | PK | |
| negocio_id | FK | |
| armazem_id | FK | de onde saiu o stock |
| cliente_id | FK nullable | |
| conta_id | FK nullable | preenchido se veio de uma conta (bottlestore) |
| tipo_venda | enum | `direta` (ferragem/balcão), `conta_fechada` (bottlestore) |
| subtotal, desconto_total, total | decimal | |
| status | enum | `paga`, `divida`, `parcial`, `cancelada` |
| criado_em | timestamp | |
| utilizador_id | FK | |

#### `itens_venda`
| id, venda_id, produto_id, unidade_produto_id, quantidade, preco_unitario, custo_unitario_no_momento, desconto, subtotal |
> Guardar `custo_unitario_no_momento` é o que permite calcular **lucro real por venda** mesmo que o custo do produto mude depois.

#### `pagamentos`
| id, venda_id, metodo (`dinheiro`,`mpesa`,`emola`,`cartao`,`transferencia`), valor, criado_em |

#### `dividas`
| id, cliente_id, venda_id, valor_original, valor_pago, valor_em_aberto, status (`aberta`,`paga`,`parcial`), criado_em |

#### `pagamentos_divida`
| id, divida_id, valor, metodo, criado_em, utilizador_id |

#### `utilizadores`
| id, negocio_id (nullable se acesso a ambos), nome, username, password_hash, role (`admin`,`gerente`,`operador_pos`), ativo |

### 3.5 Nota de design: "Contas" também servem a ferragem

Para manter **um único modelo**, a venda direta da ferragem pode ser modelada como uma `conta` que abre e fecha na mesma transação (transparente para o utilizador — ele nunca vê o conceito de "conta" no frontend da ferragem, só vê "Venda"). Isto evita duplicar lógica de cálculo de totais, descontos e geração de `vendas`/`itens_venda`. O frontend da ferragem simplesmente chama `AbrirConta → AdicionarItens → FecharConta(pagamento imediato)` num único fluxo de ecrã, sem mostrar o conceito de conta ao operador.

---

## 4. Funcionalidades Detalhadas

### 4.1 Gestão de Produtos
- CRUD de produtos, com categoria, tipo (`simples`, `composto`, `semi_elaborado`), unidade base, unidades de venda/compra (caixa, pack, unidade, dose), stock mínimo.
- Upload de imagem do produto (para grelha visual no POS).
- Código de barras por unidade (permite ler tanto a caixa como a unidade).
- Ativar/desativar produto sem apagar histórico.
- Duplicar produto (útil para variantes: ex. mesma cerveja, tamanhos diferentes).
- Alertas de stock mínimo por armazém (não aplicável a produtos `composto`).

### 4.2 Fichas Técnicas / Receitas (produtos compostos e semi-elaborados)
- Ecrã de "Ficha Técnica" ao editar um produto `composto` ou `semi_elaborado`: lista de ingredientes (produtos `simples`/`semi_elaborado`), cada um com quantidade na unidade base e percentual de perda opcional.
- Cálculo automático do **custo teórico** do produto composto (soma do custo dos ingredientes ao custo médio atual), mostrado lado a lado com o preço de venda definido — dá visibilidade imediata da margem antes mesmo de vender.
- Simulador "quanto consigo vender com o stock atual": mostra quantas unidades do composto são possíveis de preparar com o stock disponível dos ingredientes (útil para saber se o prato deve sair do menu do dia).
- Registo de **produção em lote** para semi-elaborados (movimento `producao`): indicar quantidade a produzir, sistema calcula e consome os ingredientes automaticamente, cria stock do semi-elaborado.
- Duplicar ficha técnica (útil para variantes: "Caipirinha de Vodka" vs "Caipirinha de Gin", só muda um ingrediente).

### 4.3 Gestão de Armazéns
- CRUD de armazéns, associados a um negócio, com tipo (`principal`, `ponto_venda`, `deposito`).
- Visão de stock atual por armazém e por produto.
- Possibilidade de criar quantos armazéns forem necessários (ex: "Arrecadação", "Frigorífico 2").

### 4.4 Entrada de Stock (Compra)
**Fluxo:**
1. Selecionar armazém de destino.
2. Selecionar produto → escolher unidade de compra (ex: Caixa) → indicar quantidade e valor total pago (ou preço unitário).
3. Sistema calcula custo por unidade base e atualiza custo médio ponderado do produto.
4. Opcional: associar fornecedor e número de fatura/referência.
5. Confirmar → gera `movimento_stock` tipo `entrada`.

### 4.5 Transferência entre Armazéns
**Fluxo:**
1. Selecionar armazém origem e destino.
2. Selecionar produto(s) e quantidade (em qualquer unidade configurada).
3. Confirmar → gera **um movimento** com origem e destino preenchidos (ou dois movimentos ligados por `referencia_id`, dependendo da implementação — recomenda-se um único registo com ambos os campos preenchidos, mais simples de auditar).
4. Stock desce no armazém origem e sobe no destino, atomático (transação SQL).

### 4.6 Registo de Perdas
**Fluxo:**
1. Selecionar armazém, produto, quantidade.
2. Motivo obrigatório (dropdown: quebra, validade expirada, roubo, dano; + campo livre).
3. Confirmar → `movimento_stock` tipo `perda`, sem `armazem_destino`.

### 4.7 Consumo Interno
**Fluxo:** idêntico a perdas, mas tipo `consumo_interno` (ex: cerveja oferecida a funcionário, material usado na própria loja). Motivo/observação obrigatória. Útil para reconciliar por que o stock não bate com vendas.

### 4.8 Inventário
**Fluxo:**
1. Iniciar novo inventário para um armazém → sistema regista `quantidade_sistema` de cada produto no momento.
2. Operador conta fisicamente e insere `quantidade_contada` (pode inserir em qualquer unidade — ex: "3 caixas e 5 unidades" — sistema converte e soma).
3. Sistema calcula diferenças automaticamente.
4. Ao **fechar o inventário**, gera `movimentos_stock` tipo `ajuste_inventario` para cada diferença (positiva ou negativa), atualizando o stock real.
5. Relatório de divergências fica guardado para histórico/auditoria.

### 4.9 Clientes
- CRUD de clientes (nome, telefone, NIF opcional, limite de crédito opcional).
- Cliente "Anónimo" sempre disponível por omissão nas vendas.
- Histórico de compras e dívidas por cliente.
- Bloqueio automático opcional de nova dívida se limite de crédito for excedido (alerta, não obrigatório bloquear).

### 4.10 Vendas / Contas

**Ferragem (venda direta):**
1. Operador pesquisa/seleciona produtos (grelha ou código de barras).
2. Ajusta quantidades/unidade.
3. Seleciona cliente (opcional, por omissão "anónimo").
4. Aplica desconto (se permitido pela role).
5. Escolhe forma de pagamento (pode ser múltipla — ex: parte dinheiro, parte Mpesa).
6. Confirma → gera venda + baixa de stock automática do armazém de ponto de venda.
7. Opção de imprimir/gerar recibo (PDF).

**Bottlestore (contas abertas):**
1. **Abrir conta**: identificador livre (nome, "grupo", número) + cliente opcional.
2. **Lançar consumo**: ao longo da noite, adicionar produtos à conta (múltiplas vezes, por diferentes pedidos).
3. Ver lista de **contas abertas** em tempo real (ecrã principal do operador).
4. **Pedir conta / Fechar conta**: mostra resumo de tudo consumido, total a pagar.
5. Fecho: pagamento total (dinheiro/Mpesa/etc.) **ou** registar como **dívida** (só permitido se cliente estiver identificado, não anónimo).
6. Ao fechar, sistema gera `venda` + `itens_venda` a partir dos `itens_conta`, baixa stock, e (se dívida) cria registo em `dividas`.
7. Opção de **cancelar conta** (com justificação, para contas abertas por engano).
8. Opção de **transferir itens entre contas** (útil se o grupo se dividir).

**Venda de produtos compostos (aplica-se a ambos os negócios):**
Quando um item vendido é `composto` (ex: "Caipirinha de Vodka" numa conta da bottlestore) ou envolve receita (ex: um restaurante anexo à ferragem, se aplicável), o passo de baixa de stock descrito acima expande automaticamente a ficha técnica do produto e desconta cada ingrediente real — ver fluxo detalhado em 3.3.1. Para o operador, o processo no ecrã é idêntico ao de vender qualquer outro produto (seleciona "Caipirinha de Vodka" na grelha); a expansão da receita é totalmente transparente.

### 4.11 Dívidas
- Lista de dívidas em aberto por cliente.
- Registo de pagamento parcial ou total de dívida (independente de nova venda).
- Alertas de dívidas antigas (ex: > 30 dias).

### 4.12 Relatórios (ver secção 6 em detalhe)

### 4.13 Utilizadores e Permissões
| Role | Permissões |
|---|---|
| **Admin** | Acesso total: produtos, armazéns, utilizadores, relatórios de lucro, configurações |
| **Gerente** | Stock, inventário, relatórios (sem gestão de utilizadores), pode dar descontos maiores |
| **Operador POS** | Só abre/fecha contas ou vendas diretas, sem acesso a custos/lucros, desconto limitado ou nulo |

---

## 5. Fluxos Detalhados por Perfil

### 5.1 Fluxo — Administração Universal

```
Login (Admin)
  │
  ├─► Dashboard geral (ambos os negócios lado a lado, ou seletor de negócio)
  │      - Vendas hoje / semana / mês (ambos os negócios)
  │      - Alertas de stock mínimo
  │      - Contas abertas neste momento (bottlestore)
  │
  ├─► Produtos → CRUD, unidades, categorias
  ├─► Armazéns → CRUD, ver stock por armazém
  ├─► Stock → Entradas / Transferências / Perdas / Consumo / Inventário
  ├─► Clientes → CRUD, ver dívidas, histórico
  ├─► Vendas → histórico completo, filtros, detalhe de cada venda
  ├─► Relatórios → gerar e exportar (Excel/PDF)
  └─► Utilizadores → CRUD, roles, reset password
```

### 5.2 Fluxo — POS Bottlestore

```
Login (Operador)
  │
  ├─► Ecrã principal: Lista de Contas Abertas (cards, tipo comandas)
  │       [+ Nova Conta]
  │
  ├─► Abrir Conta → identificador + cliente opcional → Conta criada
  │
  ├─► Selecionar conta aberta → Ecrã da Conta
  │       - Grelha de produtos (categorias em abas: Cervejas, Destilados, Refrigerantes...)
  │       - Adicionar item → soma à conta, atualiza total em tempo real
  │       - Remover/editar item (se permitido)
  │       - Botão [Pedir Conta / Fechar]
  │
  └─► Fechar Conta
          - Resumo dos itens e total
          - Escolher: Pagar agora (dinheiro/Mpesa/cartão, split possível)
                       ou Registar Dívida (exige cliente identificado)
          - Confirmar → gera venda, imprime recibo (opcional)
          - Conta passa a "fechada", sai da lista de abertas
```

### 5.3 Fluxo — POS Ferragem

```
Login (Operador)
  │
  ├─► Ecrã principal: Ponto de Venda
  │       - Pesquisa/leitor de código de barras
  │       - Grelha de produtos por categoria
  │       - Carrinho lateral com itens, quantidades, subtotal
  │
  ├─► Selecionar cliente (opcional — omissão: anónimo)
  ├─► Aplicar desconto (se permitido)
  ├─► Finalizar Venda
  │       - Forma(s) de pagamento
  │       - Confirmar → baixa stock, gera venda
  │       - Imprimir recibo / enviar por referência
  └─► Histórico de vendas do dia (consulta rápida, reimpressão)
```

### 5.4 Fluxo — Inventário (partilhado, acedido via Admin ou Gerente)

```
Iniciar Inventário (escolher armazém)
  │
  ├─► Sistema mostra lista de produtos com quantidade_sistema
  ├─► Operador insere quantidade_contada (por produto)
  ├─► Sistema calcula diferenças em tempo real
  ├─► Rever divergências (destacar produtos com diferença)
  └─► Fechar Inventário
          - Gera ajustes de stock automaticamente
          - Guarda relatório de divergências (auditoria)
```

---

## 6. Relatórios

Todos os relatórios devem suportar: **filtro por período** (dia/semana/mês/personalizado), **filtro por armazém**, **filtro por negócio** (na visão admin universal), e **exportação em Excel (.xlsx) e PDF**.

| Relatório | Conteúdo | Notas |
|---|---|---|
| **Vendas por período** | Total vendido, nº de vendas, ticket médio, por dia/semana/mês/período customizado | Gráfico de linha/barras + tabela |
| **Lucro por período** | Receita − custo (usando `custo_unitario_no_momento`), margem % | Pode segmentar por produto/categoria |
| **Vendas por produto** | Ranking de produtos mais vendidos (quantidade e valor), incluindo compostos | Útil para decisão de compra |
| **Consumo teórico vs real de ingredientes** | Para produtos `simples` usados em receitas: quanto a receita previu consumir (baseado nas vendas) vs quanto realmente saiu do stock | Cruza com inventário — revela perdas de corte/porção acima do esperado |
| **Vendas por método de pagamento** | Dinheiro vs Mpesa vs cartão etc. | Para reconciliação de caixa |
| **Stock atual** | Quantidade por produto/armazém, valor de stock (custo) | Alertas de stock mínimo destacados |
| **Movimentos de stock** | Extrato de entradas/saídas/transferências/perdas num período | Auditoria |
| **Perdas e consumo interno** | Total perdido/consumido, valorizado ao custo | Indicador de "fugas" |
| **Contas abertas / fechadas** (bottlestore) | Histórico de contas, tempo médio aberto, valor médio por conta | |
| **Dívidas em aberto** | Por cliente, com idade da dívida | Alerta de dívidas antigas |
| **Fecho de caixa diário** | Total vendido, total recebido por método, diferenças esperadas vs contadas | Para fecho de turno |
| **Comparativo entre negócios** | Bottlestore vs Ferragem lado a lado (admin universal) | Visão executiva |

### 6.1 Exportação Excel/PDF
- **Excel**: usar `excelize` (Go) — gera `.xlsx` com folhas separadas por relatório, formatação de cabeçalho, totais a negrito, fórmulas de soma quando aplicável.
- **PDF**: usar `gofpdf` para relatórios tabulares simples, ou gerar HTML (com o mesmo template usado no ecrã) e converter via `go-wkhtmltopdf`/`chromedp` (headless Chrome) para PDFs com melhor layout — recomendado para recibos e relatórios com gráficos.
- Botão **[Exportar]** consistente em todos os ecrãs de relatório, com escolha do formato.

---

## 7. Design Visual — Cores, Identidade e Imagens

Cada frontend deve ter identidade própria (o operador sabe imediatamente em que loja está), mas o Admin deve ser neutro e consistente.

### 7.1 Bottlestore — tema "noite / bar"
- **Paleta:**
  - Fundo principal: `#14121A` (quase preto, tom roxo-carvão)
  - Superfícies/cards: `#1F1B29`
  - Cor de destaque (accent): `#D4A24E` (dourado envelhecido — remete a bebidas/whisky)
  - Cor secundária: `#7B3F61` (vinho/roxo, para categorias/tags)
  - Texto principal: `#F2ECE4` (branco quente)
  - Estados: sucesso `#4CAF7D`, alerta `#E0A62B`, erro `#D9534F`
- **Tipografia:** fonte serif ou display discreta nos títulos (ex: "Playfair Display" ou "DM Serif Display") + sans-serif limpa no corpo (ex: "Inter").
- **Iconografia/imagens:** ícones outline finos (copo, garrafa, gelo), fotos de produtos com fundo escuro, cantos arredondados nos cards de conta (parecem "comandas"). Ícone do app: taça/garrafa estilizada em dourado sobre fundo escuro.
- **Sensação geral:** elegante, discreto, "lounge noturno" — não gritante, prioriza legibilidade em ambiente de pouca luz.

### 7.2 Ferragem — tema "industrial / prático"
- **Paleta:**
  - Fundo principal: `#F4F2ED` (cinza-areia claro, ambiente de trabalho)
  - Superfícies/cards: `#FFFFFF` com borda `#E3DFD6`
  - Cor de destaque (accent): `#E8611C` (laranja industrial — remete a ferramentas/EPI)
  - Cor secundária: `#2C3539` (grafite/aço)
  - Texto principal: `#22262B`
  - Estados: sucesso `#3E8E5A`, alerta `#E0A62B`, erro `#C1443D`
- **Tipografia:** sans-serif robusta e muito legível (ex: "Inter" ou "IBM Plex Sans"), pesos fortes em títulos e preços (importante em POS de balcão, leitura rápida).
- **Iconografia/imagens:** ícones sólidos (caixa, chave inglesa, parafuso), grelha de produtos densa e funcional, cantos retos ou levemente arredondados (linguagem mais "utilitária" que a bottlestore). Ícone do app: martelo/chave cruzados em laranja sobre fundo grafite.
- **Sensação geral:** direto, rápido, sem distrações — otimizado para velocidade de venda no balcão.

### 7.3 Admin Universal — tema neutro "profissional"
- **Paleta:**
  - Fundo: `#F7F8FA`
  - Superfícies: `#FFFFFF`
  - Accent primário: `#2F6FED` (azul confiança, usado em botões de ação)
  - Accent secundário: `#0EA5A5` (verde-azulado, para indicadores positivos/gráficos)
  - Texto: `#1A1D23`
  - Um pequeno indicador de cor por negócio nas listagens: tag dourada `#D4A24E` para itens da Bottlestore, tag laranja `#E8611C` para itens da Ferragem — assim o admin identifica de relance a que negócio pertence cada registo, mesmo numa visão combinada.
- **Tipografia:** "Inter" em todo o admin, priorizando densidade de informação (tabelas, filtros, gráficos).
- **Iconografia:** ícones outline neutros (biblioteca única, ex: Lucide/Feather), gráficos com paleta consistente (accent azul para valores, verde para lucro, vermelho para perdas/dívidas).

### 7.4 Consistência entre temas
- Os três frontends partilham a mesma **grelha de espaçamento** (ex: base 4px/8px) e os mesmos **componentes estruturais** (botões, inputs, tabelas), apenas trocando tokens de cor/tipografia via CSS variables — o que facilita manter os três a partir da biblioteca `frontend-shared`.
- Sugestão técnica: implementar como **temas CSS variables** (`--color-bg`, `--color-accent`, `--color-surface`...) definidos por app, aplicados no elemento raiz de cada frontend, permitindo reutilizar 90% dos componentes React entre bottlestore/ferragem/admin.

---

## 8. Arquitetura Técnica (Wails + Uber Fx)

### 8.1 Comunicação Go ↔ React: bindings nativos do Wails
Sem servidor HTTP intermédio. Cada `service` Go é exposto ao frontend através de um struct `App`, cujos métodos o Wails torna acessíveis diretamente em TypeScript (tipos gerados automaticamente via `wails generate`):

```go
// backend/api/app.go
type App struct {
    ProdutoService   *usecase.ProdutoService
    StockService     *usecase.StockService
    ContaService     *usecase.ContaService
    RelatorioService *usecase.RelatorioService
}

func NewApp(p *usecase.ProdutoService, s *usecase.StockService, c *usecase.ContaService, r *usecase.RelatorioService) *App {
    return &App{p, s, c, r}
}

// Métodos expostos ao frontend via bindings Wails
func (a *App) ListarProdutos(negocioID int) ([]domain.Produto, error) {
    return a.ProdutoService.Listar(negocioID)
}
func (a *App) CriarProduto(p domain.Produto) (int, error) {
    return a.ProdutoService.Criar(p)
}
```
```ts
// frontend, chamada direta e tipada, sem fetch/REST
import { ListarProdutos } from "../wailsjs/go/api/App"
const produtos = await ListarProdutos(1)
```

### 8.2 Injeção de dependências com Uber Fx
O backend é um **único código-base**, agora usado por **um único executável de loja** (genérico, configurável) mais o executável de admin. O Fx organiza a montagem do grafo de dependências (repositórios → services → `App`) num módulo central, evitando repetir "wiring" manual em cada `main.go`.

```go
// backend/di/modules.go
var CoreModule = fx.Options(
    fx.Provide(repository.NewSQLiteConn),
    fx.Provide(repository.NewProdutoRepo),
    fx.Provide(repository.NewStockRepo),
    fx.Provide(repository.NewContaRepo),
    fx.Provide(repository.NewClienteRepo),
    fx.Provide(repository.NewVendaRepo),
    fx.Provide(repository.NewConfigRepo),
    fx.Provide(usecase.NewConfigService),
    fx.Provide(usecase.NewProdutoService),
    fx.Provide(usecase.NewStockService),
    fx.Provide(usecase.NewContaService),
    fx.Provide(usecase.NewVendaService),
    fx.Provide(usecase.NewFichaTecnicaService),
    fx.Provide(usecase.NewRelatorioService),
    fx.Provide(usecase.NewExportService),
    fx.Provide(api.NewApp),
)
```

**Uso como *builder*, não como *runtime*:** o Fx é pensado normalmente para correr `fx.Run()` e ficar "vivo" a gerir o ciclo de vida da aplicação. Aqui, quem controla esse ciclo de vida é o próprio `wails.Run()` — por isso usa-se `fx.Populate` só para montar o grafo uma vez e extrair o `*App` já pronto, sem manter o container Fx em execução:

```go
// cmd/app/main.go — único executável, genérico
func main() {
    var wailsApp *api.App

    fx.New(
        CoreModule,
        fx.Provide(func() config.DBPath { return config.ResolveLocalDBPath() }), // lê config.json local
        fx.Populate(&wailsApp),
    )

    err := wails.Run(&options.App{
        Title:            wailsApp.ConfigService.NomeNegocio(), // dinâmico, lido da configuração
        Width:            1280,
        Height:           800,
        Bind:             []interface{}{wailsApp},
        // AssetServer, OnStartup (dispara o wizard se config.json não existir), etc.
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

Se `config.json` não existir (primeira execução), o `OnStartup` do Wails direciona o frontend para o **wizard de configuração** (secção 9.2) em vez do ecrã normal da app; ao concluir, o wizard grava a configuração e recarrega a app já no modo definido.

### 8.3 Um único executável de loja + executável de Admin
- `cmd/app/main.go` — executável genérico, instalado em qualquer loja; o tipo de negócio e módulos ativos vêm de `config.json`, não de flags de build.
- `cmd/admin/main.go` — executável de administração universal, capaz de apontar para múltiplas bases `.db` (potencialmente de tipos de negócio diferentes).
- O `domain`/`usecase`/`repository` é **100% partilhado** entre os dois — só muda a configuração injetada e o frontend embutido.
- Adicionar suporte a um novo tipo de negócio resume-se a acrescentar módulos/tabelas opcionais (secção 9.4) e uma opção no wizard — não implica criar um novo executável.

### 8.4 Base de dados por loja
- Cada loja tem o seu `nome-da-loja.db`, com o **mesmo schema completo** para todos os tipos de negócio — módulos inativos simplesmente não têm as suas tabelas/ecrãs usados na prática (ex: uma ferragem não usa `contas`/`dividas`, mas o schema base existe da mesma forma, tal como descrito em 3.5).
- Migrações geridas com `golang-migrate` ou `goose`, versionadas junto ao binário; migrations específicas de módulo (ex: `lotes_produto`, `receitas_medicas`) só são aplicadas quando o respetivo módulo é ativado no wizard.

### 8.5 Admin universal — como agregar negócios de tipos diferentes
Sem servidor HTTP, o admin universal corre como executável Wails separado **na mesma máquina/rede local** das lojas, com acesso de leitura/escrita direto aos ficheiros `.db` de cada loja (SQLite permite abrir múltiplas bases na mesma app Go):

```go
// cmd/admin/main.go
fx.New(
    CoreModule,
    fx.Provide(func() []config.DBPath {
        return []config.DBPath{"mercearia.db", "bottlestore.db", "farmacia.db"}
    }),
    fx.Populate(&wailsApp),
)
```

Os repositórios/services do admin trabalham sobre múltiplas conexões SQLite (uma por loja), agregando os dados nos relatórios comparativos (secção 6, "Comparativo entre negócios") — mesmo que cada loja tenha módulos diferentes ativos, porque o core partilhado (produtos, stock, vendas) é sempre o mesmo.

> **Nota de evolução futura:** se um dia as lojas ficarem fisicamente separadas e precisares de acesso remoto em tempo real (não só na mesma rede local), esta é a única situação em que faria sentido reintroduzir uma camada de transporte HTTP (ex: Gin) sobre o `CoreModule` — que já está desacoplado do transporte, tornando essa adição incremental e não uma reescrita.

---

## 9. Arquitetura Genérica Multi-Negócio (Configuração em Runtime)

### 9.1 Princípio

Em vez de decidir o tipo de negócio em **tempo de compilação** (um build por negócio), a decisão passa a ser feita em **tempo de execução**, no primeiro arranque da app, através de um wizard de configuração. O core de dados (produtos, stock, vendas, clientes) é sempre o mesmo — o que muda é que **módulos opcionais** ficam ativos ou inativos, e que **tema visual** é carregado.

| Tipo de negócio | Módulos tipicamente ativos |
|---|---|
| **Mercearia** | POS direto, venda a granel/peso |
| **Ferragem / Loja de materiais** | POS direto, unidades/embalagens (caixa/pack/unidade) |
| **Farmácia** | POS direto, lotes e validade (FEFO), receitas médicas |
| **Bottlestore (bebidas + comida)** | Contas abertas, dívidas, fichas técnicas (produtos compostos) |
| **Personalizado** | Qualquer combinação, escolhida manualmente no wizard |

### 9.2 Wizard de primeiro arranque

```
1º arranque da app
   │
   ├─► Ecrã: "Bem-vindo — configure o seu negócio"
   │       Nome do negócio: [_______]
   │       Tipo de negócio:
   │         ( ) Mercearia
   │         ( ) Ferragem / Loja de materiais
   │         ( ) Farmácia
   │         ( ) Bottlestore (bebidas + comida)
   │         ( ) Personalizado (escolher módulos manualmente)
   │
   ├─► Se "Personalizado": checklist de módulos
   │       [x] Venda direta (POS)
   │       [ ] Contas abertas (tipo restaurante/bar)
   │       [ ] Fichas técnicas (produtos compostos: pratos/bebidas)
   │       [ ] Lotes e validade
   │       [ ] Receitas médicas
   │       [ ] Venda a granel/peso
   │       [ ] Gestão de dívidas de clientes
   │
   └─► Confirmar → grava configuração (config.json + tabela `configuracoes`)
          → aplica seed de categorias/produtos exemplo do tipo escolhido
          → app pronta a usar
```

Esta escolha é feita **antes** de a base de negócio ser considerada "pronta", porque decide que migrations/seeds adicionais aplicar (ex: tabelas de lotes só fazem sentido preenchidas se `modulo_lotes_validade = true`).

### 9.3 Onde e como a configuração é gravada

**`config.json` local**, lido no arranque, antes de qualquer outra inicialização:
```json
{
  "negocio_nome": "Mercearia da Dona Ana",
  "tipo_negocio": "mercearia",
  "modulos_ativos": ["pos_direto", "venda_granel"],
  "tema": "mercearia",
  "instalado_em": "2026-09-16T10:00:00Z"
}
```
E espelhada na tabela `configuracoes` (3.2), para que qualquer `service` do backend possa consultar os módulos ativos via query normal, sem depender de reler o ficheiro.

### 9.4 Extensões de modelo de dados por tipo de negócio

O core (produtos, stock, vendas, clientes, `fichas_tecnicas`) mantém-se igual para todos os tipos. Módulos adicionam **tabelas próprias**, só relevantes se o respetivo módulo estiver ativo:

**Farmácia — lotes, validade e receitas médicas:**
```
lotes_produto
  id, produto_id, numero_lote, data_validade, quantidade, armazem_id, custo_unitario
```
> Ativa rastreio por lote e **FEFO** (First-Expire-First-Out — vender primeiro o que expira mais cedo), com alertas de validade próxima. Complementa/substitui a `stock_atual` simples para produtos com `lote_controlado = true`.

```
receitas_medicas
  id, venda_id, cliente_id, medico, numero_receita, validado_por, criado_em
```
> Nome distinto de `fichas_tecnicas` (secção 3.3) para evitar ambiguidade — uma é ficha técnica de produto composto, a outra é prescrição médica.

**Mercearia — venda a granel:**
- Não exige tabela nova: reutiliza `unidades_produto` (produto com `unidade_base = kg`, unidade de compra "Pacote 1kg", venda fracionada livre, ex: 0.350 kg). Só exige UI que aceite input de peso livre — preparado para, no futuro, ligar a uma **balança eletrónica** via porta série/USB.

### 9.5 Menu e ecrãs de venda dinâmicos no frontend

O frontend único lê os módulos ativos (via `ConfigService`) e renderiza o menu e o fluxo de venda condicionalmente:

```tsx
const menu = [
  { label: "Produtos", show: true }, // sempre visível
  { label: "Fichas Técnicas", show: modulos.includes("fichas_tecnicas") },
  { label: "Contas Abertas", show: modulos.includes("contas_abertas") },
  { label: "Lotes e Validade", show: modulos.includes("lotes_validade") },
  { label: "Receitas Médicas", show: modulos.includes("receitas_medicas") },
  { label: "Dívidas", show: modulos.includes("dividas") },
];
```
No ecrã de venda: se `contas_abertas` estiver ativo, mostra-se a lista de contas (bottlestore); caso contrário, vai direto ao carrinho de venda (mercearia/ferragem/farmácia) — reaproveitando o mecanismo já descrito em 3.5 (a venda direta é internamente uma "conta" que abre e fecha na mesma transação).

### 9.6 Tema visual por tipo de negócio

Estende a lógica de temas CSS variables (secção 7) com mais duas paletas, carregadas dinamicamente consoante `tema` na configuração:

| Tipo | Accent sugerido | Sensação |
|---|---|---|
| Mercearia | Verde `#3E8E5A` sobre fundo branco | Fresco, confiança, produtos do dia-a-dia |
| Farmácia | Azul `#3A8FD4` / verde-azulado `#0EA5A5` sobre fundo branco | Clínico, limpo, confiável |
| Ferragem | Laranja `#E8611C` sobre cinza-areia | Industrial (já definida em 7.2) |
| Bottlestore | Dourado `#D4A24E` sobre fundo escuro | Noite/lounge (já definida em 7.1) |

### 9.7 Impacto na arquitetura Wails/Fx

Esta abordagem **simplifica** a arquitetura de builds descrita na secção 8:

- Deixa de haver `cmd/bottlestore/main.go` e `cmd/ferragem/main.go` separados — passa a existir **um único `cmd/app/main.go`**, que lê `config.json` no arranque e o resto da app reage a essa configuração.
- O `CoreModule` do Fx mantém-se igual; acrescenta-se um `fx.Provide(usecase.NewConfigService)`, consultado por outros services e pelo frontend para saber que módulos estão ativos.
- **Um único instalador** distribuído a todos os clientes (mercearia, farmácia, ferragem, bottlestore) — só há uma versão da app para manter e atualizar.
- O **Admin universal** (secção 8.5) mantém-se igual na forma, mas agora pode agregar lojas de **tipos diferentes** — o dashboard comparativo passa a mostrar, por exemplo, mercearia + farmácia + bottlestore lado a lado, já que todas partilham o mesmo core de dados.

---

## 10. Requisitos Não Funcionais
- **Offline-first**: a app deve funcionar 100% sem internet (base local), já que é o cenário típico de loja física.
- **Performance**: operações de POS (adicionar item, fechar conta) devem responder em < 200ms percecionados.
- **Auditoria**: todo movimento de stock e venda regista `utilizador_id` e `timestamp` — nunca apagar registos, usar `cancelada`/`estornada` como estado.
- **Backup**: rotina automática de cópia do ficheiro `.db` (ex: diária, para pasta local ou pen drive/cloud simples).
- **Impressão**: suporte a impressora térmica de recibos (ESC/POS) como extensão futura, além de PDF/Excel.

---

## 11. Roadmap Sugerido (fases de desenvolvimento)

1. **Fase 1 — Core:** modelo de dados genérico, produtos, armazéns, stock (entrada/transferência/perda/consumo), sem UI polida.
2. **Fase 2 — Wizard e configuração:** ecrã de primeiro arranque, `ConfigService`, menu dinâmico por módulos.
3. **Fase 3 — POS direto:** fluxo de venda direta completo (mercearia/ferragem/farmácia) + relatórios básicos de vendas.
4. **Fase 4 — Bottlestore:** contas abertas, fecho, dívidas.
5. **Fase 5 — Fichas Técnicas:** produtos compostos, semi-elaborados, produção em lote.
6. **Fase 6 — Módulos de nicho:** lotes/validade e receitas médicas (farmácia), venda a granel (mercearia).
7. **Fase 7 — Clientes e Dívidas** completos, com relatórios de crédito.
8. **Fase 8 — Relatórios avançados + Exportação Excel/PDF** completos.
9. **Fase 9 — Admin universal** com dashboard comparativo multi-loja, multi-tipo.
10. **Fase 10 — Polimento visual** (temas por tipo de negócio, ícones, imagens) e testes com utilizadores reais.

---

*Fim do documento.*
