# Plano de Implementação — Sistema de Gestão de Retalho Multi-Negócio

> Baseado em `SDD_Gestao_Bottlestore_Ferragem.md` (v2.0) e nas telas aprovadas em `design/Sistema Gestao Retalho - Direcao Final.dc.html`: wizard passo 2 (`1b`), dashboard admin multi-loja (`1c`), bottlestore — contas abertas (`2a`), bottlestore — conta + fecho (`2b`), PDV ferragem (`2c`).

---

## 0. Decisões de arquitetura a fixar antes da Fase 0

O SDD deixa duas escolhas em aberto ("SQL + sqlc/gorm", secção 8.2/2.2). Proponho fixar assim, revisível:

| Decisão | Escolha proposta | Porquê |
|---|---|---|
| Camada de dados Go | **sqlc** (gera código a partir de SQL) em vez de GORM | Schema é relacional e estável (secção 3), queries de stock/custo médio são melhor controladas escritas à mão; sqlc dá type-safety sem overhead de ORM |
| Driver SQLite | `modernc.org/sqlite` (puro Go, sem CGO) | Simplifica build cross-platform do Wails (Windows/Linux/Mac) sem toolchain C por plataforma |
| Migrations | `golang-migrate` | Já sugerido no SDD 8.4; migrations "core" sempre aplicadas + migrations "de módulo" aplicadas condicionalmente pelo wizard (9.4) |
| Estilo frontend | CSS variables por tema (`--color-bg`, `--color-accent`...) + CSS puro/CSS Modules, **sem Tailwind** | O design já foi feito em estilos inline com tokens de cor por tema (secção 7.4 do SDD); manter esse vocabulário evita duas linguagens de estilo a conviver |
| Estado frontend | Context API + hooks por domínio (`useProdutos`, `useContas`...), sem Redux | Volume de estado é moderado; Wails bindings já tratam de "fetch", falta só cache/local state |
| Geração de tipos TS | `wails generate` (bindings automáticos) | Conforme secção 8.1 do SDD — zero REST, zero duplicação manual de tipos |

Estas escolhas devem ser confirmadas contigo antes da Fase 0; o resto do plano assume-as.

---

## 1. Fase 0 — Scaffolding e fundação técnica

**Objetivo:** esqueleto do projeto a compilar e correr, sem funcionalidade de negócio.

- `git init` no diretório do projeto (atualmente sem controlo de versão).
- `wails init -n gestao-retalho -t react-ts` como base do `/cmd/app`.
- Estrutura de pastas conforme secção 2.2 do SDD, com um ajuste face ao original: `go:embed` não aceita `..` no caminho, por isso o frontend de cada executável tem de estar fisicamente ao lado do respetivo `main.go`, não numa pasta `/frontend` partilhada na raiz:
  ```
  gestao-retalho/
    backend/{domain,usecase,repository,api,export,config,di}   ← 100% partilhado
    cmd/app/frontend/src/{components,modules,themes,wizard,hooks,types}
    cmd/admin/frontend/pages   (Fase 9, equivalente ao /frontend-admin do SDD)
    migrations/
  ```
  Consequência prática: reutilização de componentes entre `cmd/app` e `cmd/admin` (secção 7.4 do SDD) passa a ser cópia deliberada ou, se a duplicação incomodar antes da Fase 9, extração para um pacote npm local via npm workspaces — decisão a tomar quando o admin arrancar, não antes.
- `go.mod` com Uber Fx, modernc.org/sqlite, golang-migrate, sqlc (dev tool), excelize, gofpdf.
- `backend/di/modules.go` com `CoreModule` vazio (só `NewSQLiteConn` por agora) — depois cresce a cada fase.
- `cmd/app/main.go` mínimo: `fx.Populate` + `wails.Run` a abrir uma janela em branco.
- Temas CSS: `frontend/themes/{mercearia,ferragem,farmacia,bottlestore,admin}.css` com as paletas exatas da secção 7 e 9.6 do SDD, aplicadas via `data-theme` no elemento raiz.
- Lint/format: `golangci-lint`, ESLint + Prettier, `.editorconfig`.
- CI mínima (GitHub Actions ou equivalente): `go build`, `go vet`, `npm run build` do frontend — sem deploy ainda.

**Saída:** app Wails abre janela vazia, tema aplicável, pipeline de bindings Go→TS a funcionar com um endpoint de teste (`Ping()`).

---

## 2. Fase 1 — Core: produtos, armazéns, stock

Mapeia à Fase 1 do roadmap do SDD (secção 11) e às entidades da secção 3.2.

**Backend:**
- Migrations core: `negocios`, `configuracoes`, `armazens`, `categorias`, `produtos`, `unidades_produto`, `movimentos_stock`, `stock_atual` (como view SQL — recomendado no SDD para evitar inconsistência), `inventarios`, `inventario_itens`.
- `repository`: implementações sqlc para cada tabela acima.
- `usecase`: `ProdutoService` (CRUD, duplicar produto, ativar/desativar), `StockService` (entrada, transferência, perda, consumo interno, inventário — fluxos 4.4 a 4.8 do SDD), `ArmazemService`.
- Regra de negócio central a testar bem cedo: produtos `composto` nunca aparecem em `stock_atual` (`controla_stock = false`), e todo movimento de stock é transacional (SQL transaction) para o par origem/destino em transferências.
- `api/app.go`: bindings `ListarProdutos`, `CriarProduto`, `RegistarEntradaStock`, `TransferirStock`, `RegistarPerda`, `IniciarInventario`, `FecharInventario`.

**Frontend:**
- Componentes partilhados de `frontend/components`: `ProductGrid`, `ProductForm`, `WarehouseSwitcher`, `StockMovementForm`, `Table` genérica com filtros — estes são os blocos reutilizados depois nas telas `2a`/`2b`/`2c` do design.
- Ainda sem tema de negócio aplicado (usa tema admin neutro) — é só onde as telas de gestão de stock vivem por agora.

**Saída / critério de aceitação:** dá para criar produto, dar entrada de stock em dois armazéns, transferir, registar perda, fazer um inventário completo e ver `stock_atual` a refletir tudo — sem UI polida, só funcional.

**Testes:** unitários Go para `StockService` (casos: entrada aumenta stock, transferência é atómica e falha se stock insuficiente, perda exige motivo, inventário gera `ajuste_inventario` correto para diferenças positivas e negativas).

> **Estado: concluída.** Duas notas sobre decisões tomadas durante a implementação:
> - `migrations/` não pôde ficar solta na raiz do módulo pela mesma razão do frontend na Fase 0 (`go:embed` não aceita `..`) — vive em `backend/repository/migrations/`, embutida por `backend/repository/migrate.go` e aplicada automaticamente a cada `NewSQLiteConn` (nenhum passo manual de migração a lembrar).
> - Custo médio ponderado ficou por **(produto, armazém)**, não só por produto — a tabela `stock_atual` do SDD 3.2 pareia as duas colunas explicitamente, mesmo a prosa da secção 4.4 falando só em "produto". Transferências entram no destino ao custo médio da origem; saídas (perda/consumo/venda) nunca alteram a média do que fica.
> - `utilizadores` entrou já nesta fase (não só na Fase 2) porque `movimentos_stock.utilizador_id` é `NOT NULL` desde o primeiro registo — há uma linha semente (`id=1`, sem password) até o wizard criar o admin real.
> - Frontend da Fase 1 (`cmd/app/frontend/src/FaseUm.tsx`) é deliberadamente sem tema/estilo — existe para exercitar os bindings ponta a ponta, não para o operador final usar; a previsão original era ser "substituído pelos ecrãs `1b`/`2a`/`2b`/`2c` nas fases seguintes" — **isto era impreciso**: `1b`/`2a`/`2b`/`2c` são só o wizard e os ecrãs de venda, nunca cobriram gestão de produtos/categorias/armazéns/stock em si. Sem nenhuma fase do roteiro alocada para desenhar esse ecrã, o `FaseUm.tsx` foi crescendo (uma secção nova a cada fase que precisava de validar bindings — fichas técnicas na 5, farmácia na 6, clientes na 7, relatórios na 8) e ficou por resolver até um pedido explícito do utilizador depois da Fase 10 ("único ponto de administração"). Substituído nessa altura por `Administracao.tsx` — sidebar de navegação (Produtos, Categorias, Armazéns, Stock, Inventário, Fichas técnicas/Farmácia quando o módulo respetivo está ativo, Clientes e dívidas, Relatórios), com um separador "Venda / Administração" em `App.tsx` — ver nota correspondente antes da Fase 9.

---

## 3. Fase 2 — Wizard de 1º arranque e configuração (tela `1b`)

- `usecase.ConfigService`: lê/escreve `config.json` local + espelha em `configuracoes` (secção 9.3 do SDD). `OnStartup` do Wails redireciona para o wizard se `config.json` não existir.
- Implementar a tela `1b` do design tal como desenhada: painel lateral com 4 passos (nome do negócio → tipo de negócio → módulos e armazéns → utilizador admin), pré-visualização de tema em tempo real, lista de "migrations a aplicar" dinâmica conforme o tipo escolhido.
- Ao confirmar: aplica migrations de módulo (9.4) condicionalmente, grava seed de categorias/produtos de exemplo do tipo escolhido, cria utilizador admin inicial (`utilizadores`).
- Menu dinâmico do frontend (secção 9.5 do SDD) lê `modulos_ativos` via `ConfigService` e mostra/esconde: Fichas Técnicas, Contas Abertas, Lotes e Validade, Receitas Médicas, Dívidas.
- Aplicação do tema (`tema` em `configuracoes`) no elemento raiz do frontend, carregando o CSS correspondente da Fase 0.

**Saída:** primeira execução da app cai sempre no wizard; ao terminar, abre já com o menu e tema corretos para o tipo escolhido, sem reiniciar o processo.

> **Estado: concluída.** Notas sobre decisões tomadas durante a implementação:
> - `EstaConfigurado` é decidido pela chave `wizard_concluido` em `configuracoes` (SQLite), não por `config.json` — este continua a ser gravado como espelho (SDD 9.3), mas nunca é relido pela app; como a Fase 0 já garante que a base está sempre aberta antes do frontend arrancar, não há vantagem prática em depender do ficheiro para decidir o routing.
> - "Aplica migrations de módulo condicionalmente" (9.4) ainda não tem o que aplicar: as tabelas de `lotes_produto`, `receitas_medicas`, `contas`, `fichas_tecnicas` etc. só nascem nas Fases 4-6. Por agora `configuracoes` grava só as flags `modulo_*` (todas, mesmo as `false`, para o menu dinâmico nunca ter de tratar "chave ausente" como caso especial) — o painel "migrations a aplicar" do mock da tela `1b` foi substituído por uma lista real dos módulos ativados, para o wizard nunca mostrar algo que a app ainda não faz.
> - Menu dinâmico (9.5) ainda não tem itens condicionais para mostrar/esconder — os ecrãs de Fichas Técnicas/Contas Abertas/etc. só existem a partir da Fase 4+. `ConfigService.ObterEstado` já expõe `ModulosAtivos` prontos a consumir quando esses ecrãs chegarem.
> - Passo 4 do wizard (utilizador administrador) grava username/password reais com bcrypt no utilizador semente da Fase 1 — mas ainda não há ecrã de login a pedir essas credenciais; `UtilizadorAtualID` em `api.App` continua fixo até a sessão real chegar (Fase 7 trata utilizadores/permissões por completo).
> - Bug apanhado pelos testes antes de chegar à UI: `ConfigService.FinalizarWizard` esquecia-se de propagar `NegocioID` ao criar o armazém principal, violando a FK. Corrigido movendo a atribuição de `NegocioID` para dentro de `ProdutoService.Criar`/`ArmazemService.Criar` (fonte única, não confiada a cada chamador) em vez de a repetir em cada binding do `api.App`.

---

## 4. Fase 3 — PDV direto (Ferragem / Mercearia / Farmácia) — tela `2c`

- `usecase.VendaService`, `ClienteService` (cliente anónimo por omissão), `PagamentoService` (suporta pagamento múltiplo/split, ver modal de fecho da tela `2b` que reutiliza a mesma lógica).
- Implementação fiel da tela `2c`: input de código de barras sempre focado, atalho `F2` para focar pesquisa, categorias em chips, grelha de artigos com indicador de stock por cor (cinza normal / dourado baixo / vermelho crítico — usar os limiares do `stock_minimo` por produto), carrinho lateral com `+`/`−` por linha, atalho `F12` para finalizar venda.
- Fluxo 4.10 (Ferragem) do SDD: pesquisar/selecionar → ajustar quantidade/unidade → cliente opcional → desconto (limitado por role, 4.13) → pagamento(s) → confirmar (baixa stock + gera venda) → recibo PDF.
- Nota de design 3.5: internamente a venda direta é uma `conta` que abre e fecha na mesma transação — implementar assim desde já para reaproveitar 100% da lógica de contas da Fase 4, mesmo que o operador nunca veja o conceito.
- Recibo: gerar PDF simples via `gofpdf` já nesta fase (relatórios mais ricos ficam para a Fase 8).

**Saída:** ferragem/mercearia/farmácia conseguem operar um dia de vendas completo end-to-end, incluindo histórico do dia (aba "Histórico do dia" visível na tela `2c`).

**Testes:** cálculo de subtotal/desconto/total, bloqueio de venda sem stock suficiente, `custo_unitario_no_momento` gravado corretamente em `itens_venda`.

> **Estado: concluída.** Decisões tomadas durante a implementação:
> - SDD 3.2 contradiz-se sobre o cliente anónimo: a tabela `clientes` diz "registo fixo (id=0 ou is_anonimo)", mas `contas`/`vendas` dizem "cliente_id FK nullable | null = anónimo". Fiquei com o **registo fixo** (`clientes.id=1`, `is_anonimo=true`, semeado na migration): `cliente_id` é `NOT NULL` em `contas`/`vendas`, nunca precisa de tratamento especial para NULL em relatórios/joins futuros.
> - Confirmei ao implementar o fluxo que stock só desce **ao fechar a conta**, não ao adicionar item — 4.10 (bottlestore) diz isso explicitamente ("ao fechar, baixa stock"); a venda direta (Fase 3) só passa por lá porque abre+fecha na mesma chamada, mas o mecanismo é o mesmo que a Fase 4 vai reaproveitar sem alterações.
> - Não criei uma `ContaService` pública separada — `VendaService.RegistarVendaDireta` já orquestra abrir→lançar itens→fechar→gerar venda internamente via `VendaRepository` (contas+itens_conta+vendas+itens_venda+pagamentos, todos numa única transação partilhada com o `StockRepository`). A Fase 4 (contas abertas ao longo da noite) estende isto com métodos novos no mesmo repositório, não reescreve nada.
> - Venda direta em Fase 3 **exige pagamento integral** (soma dos pagamentos == total, com tolerância de arredondamento): a tabela `dividas` só existe a partir da Fase 4/7, e o fluxo 4.10 da ferragem nunca menciona dívida (isso é só descrito no fecho de conta da bottlestore, 4.10 ponto 5).
> - Descontos limitados por role (SDD 4.13: "operador POS... desconto limitado ou nulo") ainda não são impostos — não há sessão/login real (isso é Fase 7); o campo de desconto existe e é validado (não pode ser negativo, não pode exceder o total), só não há ainda "quem" para aplicar o limite.
> - Recibo: em vez de um diálogo "guardar como", `GerarRecibo` escreve o PDF numa pasta `recibos/` ao lado da base e abre-o com `runtime.BrowserOpenURL` no leitor de PDF por omissão do sistema — mais simples e já cobre "imprimir/gerar recibo" (SDD 4.10) sem construir UI de diálogo de ficheiros ainda.
> - A grelha de produtos do ecrã de venda direta não tem os chips de categoria do mock `2c` — não há API de categorias exposta ainda (só existe o repositório, sem usecase/binding); ficou uma pesquisa por nome em vez disso. Indicador de stock por cor também ficou simplificado (só destaca `stock ≤ 0`, sem o limiar amarelo do `stock_minimo`) para não bloquear a fase nisso — ambos os refinamentos são baratos de acrescentar quando a Fase 6/10 tocar de novo neste ecrã.

---

## 5. Fase 4 — Bottlestore: contas abertas — telas `2a` e `2b`

- `usecase.ContaService`: abrir conta (identificador livre + cliente opcional), lançar consumo incremental, listar contas abertas em tempo real, transferir itens entre contas, cancelar conta com justificação, fechar conta.
- Tela `2a`: grelha de cartões de conta com KPIs no topo (nº de contas abertas, total em consumo), filtros "Todas / Com cliente / Abertas > 2h", indicador de tempo por conta com cor (normal vs alerta > 2h, replicando os tons `corTempo`/`fundoTempo` do mock).
- Tela `2b`: grelha de produtos por categoria à esquerda + comanda viva à direita + modal de fecho. O modal de fecho é o componente de pagamento **partilhado com a Fase 3** (split de métodos, cálculo de "falta atribuir" em tempo real, opção de dívida só ativa se cliente identificado — regra 4.10/4.11).
- `dividas`: ao fechar com dívida, cria registo em `dividas`; pagamento de dívida é fluxo independente de nova venda (4.11).
- Regra de bloqueio: dívida só permitida com cliente não-anónimo — validar tanto no frontend (desabilitar opção) como no backend (não confiar só na UI).

**Saída:** bottlestore consegue abrir várias contas em paralelo, lançar consumo ao longo da noite, transferir itens entre contas, e fechar com pagamento misto ou dívida.

**Testes:** conta não pode fechar duas vezes; transferência de item entre contas mantém total consistente nas duas; dívida bloqueada para cliente anónimo.

> **Estado: concluída.** Decisões tomadas durante a implementação:
> - `FecharConta` (bottlestore) e `RegistarVendaDireta` (Fase 3) partilham a mecânica de gerar `itens_venda`+movimentos de stock via uma função de pacote só (`criarItensVendaEMovimentos`, em `venda_service.go`) — extraí isto do código da Fase 3 em vez de duplicar a lógica de resolver linha→stock→custo; só a origem das linhas muda (parâmetros frescos na venda direta vs. `itens_conta` já persistidos ao longo da noite na bottlestore).
> - `AdicionarItem` não toca em stock nem o valida — só regista a linha na conta. O stock só é lido/baixado no `FecharConta`, confirmando a leitura da Fase 3 do fluxo 4.10: lançar consumo é uma operação "leve", a baixa de stock é sempre no fecho.
> - `FecharConta` calcula preço a partir de `itens_conta.preco_unitario_aplicado` (congelado no momento em que cada item foi lançado), não do preço de catálogo atual — o cliente paga o preço de quando pediu, mesmo que o preço mude a meio da noite.
> - Dívida: se os pagamentos não cobrem o total, a diferença vira `divida` — mas só se `conta.cliente_id` não for o anónimo (`domain.ErrClienteAnonimoNaoPodeDivida`). Pagamento parcial gera `status=parcial` (venda e dívida); zero pagamento gera `status=divida`. O fluxo de **pagar** uma dívida existente fica para a Fase 7 (SDD 4.11) — aqui só o registo é criado.
> - Transferência de item move a linha inteira (`itens_conta.conta_id` para a nova conta) — não fraciona quantidade entre duas contas; cobre o caso do SDD ("útil se o grupo se dividir") sem a complexidade extra de split parcial, que ninguém pediu.
> - No frontend, `2a`/`2b` não têm os filtros "Com cliente / Abertas > 2h" nem o destaque de cor por tempo do mock — ficou tudo numa lista simples ordenada por hora de abertura; e o modal de fecho não tem "imprimir recibo" como checkbox opcional, gera sempre (mesma simplificação da Fase 3).
> - Menu dinâmico (SDD 9.5) ligado pela primeira vez nesta fase: `App.tsx` só mostra o PDV direto se `pos_direto` estiver ativo, e a bottlestore se `contas_abertas` estiver ativo — nunca os dois ao mesmo tempo, porque nenhum preset do wizard ativa os dois juntos.

---

## 6. Fase 5 — Fichas técnicas, produtos compostos e semi-elaborados

Cobre secção 3.3 do SDD por completo — é a parte mais delicada do domínio.

- `fichas_tecnicas`: CRUD de receita por produto `composto`/`semi_elaborado`, com `perda_percentual` opcional por ingrediente.
- `FichaTecnicaService.CalcularCustoTeorico`: soma `quantidade_receita × custo_médio_atual_do_ingrediente`.
- `FichaTecnicaService.SimularDisponibilidade`: "quanto dá para vender com o stock atual" — usa o ingrediente mais restritivo da receita.
- Expansão de receita na venda (fluxo 3.3.1): ao vender 1x produto composto, gerar N movimentos de stock (um por ingrediente), calcular custo do item no momento da venda, guardar como um único `item_venda`/`item_conta` visível ao cliente.
- Movimento `producao` (3.3.2): produção em lote de semi-elaborados — consome ingredientes da receita do semi-elaborado, cria stock do semi-elaborado no armazém indicado. Suportar receitas em vários níveis (semi-elaborado usado como ingrediente de outro produto composto).
- Bloqueio/alerta configurável quando ingrediente não tem stock suficiente (mesma lógica de bloqueio de produto simples sem stock).
- UI: ecrã de "Ficha Técnica" ao editar produto composto (lista de ingredientes + custo teórico lado a lado com preço de venda + simulador de disponibilidade), ecrã de "Produção em lote" para semi-elaborados, botão "Duplicar ficha técnica".

**Saída:** vender "Caipirinha de Vodka" na tela `2b` desconta vodka/limão/açúcar/gelo individualmente, de forma transparente ao operador; produzir "Arroz Cozinhado" em lote cria stock consumindo arroz cru/água/sal.

**Testes:** venda de composto gera exatamente N movimentos (um por ingrediente) com sinal negativo correto; produção gera consumo + criação de stock na mesma transação; simulador de disponibilidade respeita o ingrediente mais restritivo.

> **Estado: concluída.** Decisões tomadas durante a implementação:
> - `quantidade` em `fichas_tecnicas` é sempre "por 1 unidade produzida/vendida" — mesma convenção para produto composto (por unidade vendida) e semi-elaborado (por unidade do lote produzido). O exemplo do SDD 3.3.2 ("produzir 5kg consome 1.2kg arroz cru") descreve o TOTAL de um lote de 5kg; a receita gravada é esse total dividido por 5 (0.24kg/kg). Um teste apanhou exatamente esta ambiguidade antes de chegar à UI — os valores esperados no teste estavam a assumir a convenção errada, não o código.
> - `perda_percentual` interpretado como SDD 3.3 descreve textualmente: `quantidade` na receita já é a quantidade **líquida**, e o consumo real de stock é `quantidade / (1 - perda%/100)` — sempre ≥ quantidade, nunca o inverso.
> - Só produtos `simples`/`semi_elaborado` podem ser ingredientes — um `composto` usado como ingrediente de outro composto exigiria expandir duas fichas técnicas em cascata na venda; o SDD já resolve "receitas em vários níveis" com semi-elaborados produzidos em lote (3.3.2), não com aninhamento de compostos, por isso bloqueei isso explicitamente em `DefinirReceita`.
> - `criarItensVendaEMovimentos` (partilhado entre venda direta e fecho de conta desde a Fase 4) ganhou um `if l.Composto` que desvia para `consumirFichaTecnica` — gera N movimentos tipo `venda` (um por ingrediente) em vez de 1, mas continua a gravar **um único** `item_venda` (o cliente só vê "1x Caipirinha de Vodka"). `ContaService.AdicionarItem` deixou de bloquear produtos compostos — a ficha técnica só é exigida ao FECHAR a conta, tal como o stock de qualquer item.
> - Custo por unidade vendida de um composto = soma do custo de todos os ingredientes consumidos ÷ quantidade vendida — mesma convenção de `CustoUnitarioNoMomento` "por unidade vendida" já usada para produtos simples (Fase 3).
> - `RegistarProducao` reaproveita `custoMedioPonderado` (a mesma função da Fase 1) para atualizar o custo médio do semi-elaborado produzido — produção é tratada exatamente como uma entrada de stock a um custo calculado, não como um mecanismo à parte.
> - `DefinirReceita` substitui a receita inteira (apaga tudo e regrava) em vez de reconciliar adicionar/remover/editar linha a linha — mais simples e mais seguro para um ecrã de edição, ao custo de perder o histórico de versões da receita (não pedido pelo SDD).
> - Frontend ficou em `FichasTecnicas.tsx`, integrado no scaffolding do FaseUm (mesma família funcional-sem-tema das Fases 1-4) — não há tela de design dedicada a fichas técnicas nos mocks aprovados, ao contrário das Fases 3/4.

---

## 7. Fase 6 — Módulos de nicho (farmácia e mercearia)

- **Farmácia:** tabela `lotes_produto`, lógica FEFO (vender primeiro o que expira mais cedo) na baixa de stock, alertas de validade próxima, tabela `receitas_medicas` associada à venda (médico, nº de receita, validado por).
- **Mercearia — venda a granel:** sem tabela nova (reutiliza `unidades_produto` com `unidade_base = kg`); UI precisa aceitar input de peso livre (ex: 0.350 kg) na grelha de venda. Deixar ponto de extensão comentado no código para integração futura com balança eletrónica via porta série/USB — não implementar agora, só não bloquear a interface para isso.
- Ambos os módulos só aparecem no menu/ecrãs se ativos em `configuracoes` (mesma mecânica da Fase 2).

**Saída:** farmácia vende com controlo de lote/validade e receita médica; mercearia vende produtos a peso livre.

> **Estado: concluída.** Decisões tomadas durante a implementação:
> - `lote_controlado` ficou como flag em `produtos` (não uma tabela de configuração à parte) — decide em runtime, por produto, se a entrada/venda passa pelo caminho de lotes ou pelo caminho normal. `StockService.RegistarEntrada` passou a rejeitar produtos lote-controlados explicitamente (erro claro a apontar para `FarmaciaService.RegistarEntradaComLote`) — sem isto, `stock_atual` e `lotes_produto` divergiam silenciosamente e o FEFO da venda lia lotes vazios.
> - FEFO ficou embutido na mesma função partilhada que já fazia a expansão de ficha técnica (`criarItensVendaEMovimentos`, Fase 4/5): um produto lote-controlado desvia para `consumirLoteFEFO`, que pode atravessar vários lotes numa só venda (esgota o mais próximo da validade, completa no seguinte) — gera um movimento de stock por lote tocado, cada um ao custo real desse lote, mais preciso que um custo médio único.
> - `LoteRepository` segue o mesmo padrão de composição do `VendaRepository`/`FichaTecnicaRepository` (Fase 4/5): carrega um `StockRepository` sempre ligado à mesma transação. `VendaRepository` passou a compor também um `LoteRepository` tx-scoped, pela mesma razão — decrementar um lote e gravar o movimento de stock têm de ser atómicos.
> - `receitas_medicas` ficou como anexo opcional a uma venda já criada (não um bloqueio na hora de vender) — o SDD não define uma regra de "este produto exige receita", por isso não inventei um campo/enforcement que o documento não pede; é só o registo (médico, nº receita, validado por) associado ao `venda_id`.
> - Venda a granel não precisou de nenhuma mudança no backend (quantidade já é `float64` em todo o lado desde a Fase 1) — só o carrinho do PDV passou a mostrar um input de peso livre em vez do contador +/- quando `produto.UnidadeBase` é "kg"/"g" (heurística simples, sem flag dedicada). Deixei o ponto de extensão para uma balança eletrónica comentado exatamente onde o peso entra no carrinho (`definirQuantidade` em `VendaDireta.tsx`), sem o implementar.
> - UI de farmácia (`Farmacia.tsx`) ficou na mesma família funcional-sem-tema do FaseUm/FichasTecnicas — sem tela de design dedicada nos mocks aprovados para esta fase.

## 8. Fase 7 — Clientes e dívidas completos

- CRUD completo de clientes, histórico de compras e dívidas por cliente.
- Bloqueio opcional (alerta, não obrigatório) de nova dívida acima do `limite_credito`.
- Alertas de dívidas antigas (> 30 dias) — reutilizados depois no painel "Alertas" do dashboard admin (`1c`).
- Pagamento parcial/total de dívida como fluxo independente.

> **Estado: concluída.** Decisões tomadas durante a implementação:
> - O bloqueio de `limite_credito` é mesmo só um alerta, nunca um erro: `ContaService.FecharConta` passou a devolver um terceiro valor, `AlertaLimiteCredito{Excedido, LimiteCredito, TotalEmAberto}`, computado **depois** de a venda/dívida já estar confirmada — nunca impede o fecho de conta, mesmo que a dívida resultante ultrapasse o limite do cliente. `ContaService` passou a depender diretamente de `*DividaService` (não só de repositórios) para reutilizar `VerificarLimiteCredito` sem duplicar a soma de dívidas em aberto.
> - `DividaRepository` tem o seu próprio `WithTx` independente, sem compor `StockRepository` (ao contrário de `VendaRepository`/`LoteRepository`) — pagar uma dívida nunca toca stock nem vendas, só `dividas` e a nova tabela `pagamentos_divida` (migração 000006), por isso não precisa da mesma composição transacional.
> - `RegistarPagamentoDivida` recalcula o novo `status` a partir do saldo (`aberta` → `parcial` → `paga`), com a mesma tolerância de arredondamento (0.01) já usada para pagamentos de venda; paga acima do saldo em aberto é rejeitado, tal como registar pagamento numa dívida já `paga`.
> - Não existe (nem deve existir) uma forma pela API pública de simular uma dívida "antiga" para testar o alerta de >30 dias — o teste correspondente (`TestListarDividasAntigas_RespeitaJanelaDeDias`) usa uma escrita SQL direta ao `*sql.DB` do teste só para recuar `criado_em`, técnica exclusiva de teste, nunca exposta em produção.
> - Frontend ficou em `Clientes.tsx`, integrado no scaffolding do FaseUm (mesma família funcional-sem-tema das Fases 1/5/6) — sem tela de design dedicada a clientes/dívidas nos mocks aprovados. O modal de fecho de conta da bottlestore (`Bottlestore.tsx`, tela `2a`/`2b`) passou a mostrar o alerta de limite de crédito quando excedido, sem bloquear a confirmação.

---

## 9. Fase 8 — Relatórios e exportação Excel/PDF

Implementar a tabela completa da secção 6 do SDD, todos com filtro por período/armazém/negócio e exportação:

- `RelatorioService` com um método por relatório (vendas por período, lucro por período, vendas por produto, consumo teórico vs real de ingredientes, vendas por método de pagamento, stock atual, movimentos de stock, perdas e consumo interno, contas abertas/fechadas, dívidas em aberto, fecho de caixa diário, comparativo entre negócios).
- `ExportService`: `excelize` para `.xlsx` (folhas separadas, cabeçalho formatado, totais a negrito), e para PDF avaliar entre `gofpdf` (tabulares simples) vs `chromedp`/`go-wkhtmltopdf` (recibos e relatórios com gráfico) — decisão a confirmar quando chegarmos aqui, conforme complexidade real dos relatórios com gráfico.
- Botão "Exportar" consistente em todos os ecrãs de relatório (padrão já visível no header da tela `1c`).
- O relatório "Consumo teórico vs real de ingredientes" depende da Fase 5 (fichas técnicas) e da Fase 1 (inventário) já estarem implementadas — é o relatório que fecha o ciclo de reconciliação mencionado na secção 3.3.

> **Estado: concluída**, com uma exclusão deliberada. Decisões tomadas durante a implementação:
> - "Comparativo entre negócios" (última linha da tabela do SDD 6) **não foi implementado** — só faz sentido na visão Admin universal multi-loja (Fase 9), que ainda não existe; dentro de uma única loja não há segundo negócio para comparar. Os outros 11 relatórios da tabela estão todos implementados.
> - `RelatorioRepository` é um repositório novo e só de leitura (não segue o padrão `WithTx` dos outros, porque nenhum método participa de escrita) — cada query já faz JOIN com `produtos`/`clientes` para devolver o nome, não só o ID, evitando N pedidos extra do frontend só para rotular uma tabela.
> - O filtro de armazém é opcional em SQL via `sqlc.narg`, comparado com `OR` contra colunas de tabelas diferentes consoante o relatório — isto faz o sqlc gerar o parâmetro como `interface{}` em vez de `sql.NullInt64` (não consegue inferir um único tipo quando o mesmo `?` aparece em mais que um contexto de coluna); passa-se `nil` para "todos os armazéns" ou o `int64` diretamente.
> - SQLite tem tipagem dinâmica: `COALESCE(SUM(coluna_real), 0)` devolve `int64` quando a agregação não toca nenhuma linha (o literal `0` vence) e `float64` quando soma pelo menos uma linha — todas as colunas agregadas dos relatórios passam por um `anyToFloat64` novo em `convert.go` que trata os dois casos, em vez de assumir sempre `float64` e rebentar com um `type assertion`.
> - "Consumo teórico vs real" não tem uma query SQL dedicada — o `RelatorioService` busca os itens de venda do período (`ItensVendaPorPeriodo`), expande a ficha técnica de cada produto composto vendido em Go (mesma fórmula `ConsumoReal()` da Fase 5) para o lado teórico, e cruza com `movimentos_stock` tipo=`venda` do ingrediente no mesmo período para o lado real — mais simples do que replicar a expansão de receita em SQL.
> - Bug encontrado e corrigido ao escrever o relatório de perdas: `StockService.registarSaida` (perda/consumo interno) nunca gravava `custo_unitario` no movimento — só entrada e venda gravavam. "Perdas ... valorizado ao custo" (SDD 6) ficava sempre a 0. Corrigido para gravar o `custo_medio_ponderado` do stock no momento da saída, igual ao que a venda já fazia; um teste do relatório apanhou isto antes de chegar à UI.
> - `ExportService` não existe como serviço dedicado — é só `export.Tabela{Titulo, Colunas, Linhas, Totais}` + duas funções (`GerarTabelaExcel` via `excelize`, `GerarTabelaPDF` via `go-pdf/fpdf`, A4 paisagem, sem gráficos). Decisão sobre PDF: ficou em `gofpdf`-style tabular simples, sem avaliar `chromedp`/`go-wkhtmltopdf` — nenhum dos 11 relatórios tem gráfico, só tabelas, por isso a alternativa mais pesada não se justificou.
> - O botão "Exportar" é genérico, não um por relatório: só há dois bindings no total (`ExportarTabelaExcel`/`ExportarTabelaPDF`), que recebem título/colunas/linhas/totais já formatados como string — o ecrã de relatórios já teve de formatar cada valor para a pré-visualização em tabela, por isso a exportação reutiliza exatamente essas mesmas linhas em vez de o backend reconstruir a formatação a partir dos dados tipados outra vez. Evita 22 bindings quase idênticos (11 relatórios × 2 formatos).
> - "Fecho de caixa diário" não tem "contado" gravado em lado nenhum — o modelo de dados não tem uma tabela de "caixa"/contagem física, e o SDD não pede uma. O backend só devolve o valor esperado por método (reaproveitando "vendas por método de pagamento" com a janela fechada a um dia); o campo "Contado" e a diferença são só de UI, no `Relatorios.tsx`, nunca chegam ao backend.
> - Frontend ficou em `Relatorios.tsx`, integrado no scaffolding do FaseUm (mesma família funcional-sem-tema das Fases 1/5/6/7) — sem tela de design dedicada a relatórios nos mocks aprovados. Um único componente de tabela genérica (`TabelaRelatorio`) é reutilizado pelos 11 relatórios, com os botões de exportação já ligados aos dois bindings genéricos.

---

## 8.5. Addendum (pós-Fase 10) — Administração única por loja

Fora da sequência numerada do roteiro: pedido explícito do utilizador depois da Fase 10, a confirmar o desenho pretendido da app — "um único ponto de administração onde se regista categorias, produtos, armazéns, etc. consoante o tipo de negócio escolhido, e isso ativa o módulo de ponto de venda específico escolhido na instalação". Resposta: a parte de "o tipo de negócio escolhido ativa o módulo de venda certo" já estava implementada desde a Fase 2/3/4 (wizard decide `ModulosAtivos`, `App.tsx` escolhe entre `VendaDireta`/`Bottlestore`); a parte que faltava era o "único ponto de administração" em si — existia funcionalmente, mas espalhado no scaffolding do `FaseUm.tsx` sem navegação nem separação do ecrã de venda (ver nota corrigida na Fase 1 acima).

**Backend (pequeno):**
- `CategoriaService` + bindings `ListarCategorias`/`CriarCategoria` — faltavam por completo; `CategoriaRepository` já existia desde a Fase 1 (usado internamente pelo wizard para semear categorias por preset) mas nunca teve usecase/binding próprios. `CategoriaID` já fluía ponta a ponta desde `ProdutoRepository.Criar` — só a UI para o escolher é que não existia.

**Frontend:**
- `FaseUm.tsx` foi substituído por `Administracao.tsx`: sidebar (produtos, categorias, armazéns, stock, inventário sempre visíveis; fichas técnicas só se o módulo `fichas_tecnicas` estiver ativo; farmácia só se `lotes_validade` ou `receitas_medicas` estiverem ativos; clientes e relatórios sempre visíveis) + área de conteúdo com a secção selecionada — mesmo padrão da sidebar de `cmd/admin` (Fase 9), mas usando os tokens de tema (`--color-*`) em vez de azul neutro fixo, porque esta administração vive dentro da mesma app com tema por tipo de negócio, ao contrário do `cmd/admin` que é propositadamente neutro.
- `App.tsx` ganhou um separador "Venda / Administração" no cabeçalho — só aparece se a loja tiver módulo de venda ativo (`pos_direto` ou `contas_abertas`); sem nenhum dos dois, a app abre direto em Administração.
- Removidos da UI de produção o seletor de tema manual (`theme-switcher`) e a caixa de teste "Ping() → SQLite" — ambos scaffolding explícito da Fase 0 (o próprio `useTheme.ts` já dizia "scaffolding-only theme switcher"), nunca destinados ao operador final; o tema já vem sempre corretamente de `config.Tema`.
- `ProdutosSection` e `ArmazensSection` ganharam Ativar/Desativar (os bindings já existiam desde a Fase 1/2, nunca tinham UI) e `ProdutosSection` ganhou o seletor de categoria.
- Reaproveitados sem alteração: `FichasTecnicas.tsx`, `Farmacia.tsx`, `Clientes.tsx`, `Relatorios.tsx` — já eram componentes à parte, só passaram a ser montados condicionalmente por secção em vez de sempre empilhados.

Testado com a base de desenvolvimento real (farmácia): `CriarCategoria`/`ListarCategorias` confirmados ponta a ponta (categorias "Medicamentos"/"Higiene" já semeadas pelo wizard nunca tinham sido visíveis em lado nenhum até agora), app lançada com `xwininfo` a confirmar a janela, sem erros no log.

---

## 8.6. Addendum (pós-Fase 10) — Autenticação: login único, só no cmd/admin

Esta entrada substitui uma versão anterior deste addendum (login por loja, dentro do `cmd/app`) — implementada, testada, e depois **revertida por completo** a pedido explícito do utilizador, ver abaixo. Fica só o desenho final.

Pedido original: ao arrancar `make run-app` sem configuração ainda, abrir o wizard (SDD 9.2); depois de o concluir, pedir login com o utilizador criado no wizard. Construído assim numa primeira volta — `AuthService` em `usecase`, sessão em `api.App`, `Login.tsx` em `cmd/app` — e confirmado a funcionar ponta a ponta. Nesse teste manual, o utilizador notou que a loja de dev (farmácia, já configurada de fases anteriores) continuava a abrir com esse login, e isso levou à pergunta seguinte, que mudou o desenho: **"já não existe admin para cada [loja], só deve existir 1 para todos, que é o cmd/admin — [o tipo de negócio] diferencia apenas a cor e o ponto de venda"**. Dado o peso arquitetural da mudança, a decisão exata foi confirmada por pergunta direta ao utilizador entre três opções (manter login por loja só com credenciais fixas de teste; remover o login do `cmd/app` por completo, autenticação só no `cmd/admin`; partilhar utilizadores entre lojas via gestão central) — escolhida a segunda.

**Consequência aceite conscientemente:** uma loja isolada (sem o `cmd/admin` por perto) deixa de ter uma barreira de login própria — quem abre o executável da loja entra direto. Isto troca a independência de autenticação 100% offline por loja (um dos princípios do SDD) pela simplicidade operacional de um único administrador a gerir cores/POS por loja, que foi o que o utilizador pediu explicitamente.

**Revertido em `cmd/app`:**
- `usecase.AuthService` (e teste) apagados; `api.App` perdeu `utilizadorAtualID`, `Login`/`Logout`/`SessaoAtiva`; voltou a constante fixa `UtilizadorAtualID = domain.UtilizadorAdminSeedID` nos 14 sítios que gravam autoria (vendas, movimentos de stock, contas, receitas médicas, etc.).
- `Login.tsx`/`Login.css` de `cmd/app` apagados; `App.tsx` voltou à máquina de estados simples `a-carregar → wizard → pronto`, sem estado `"login"` nem botão "Sair".
- O wizard **continua** a pedir nome/username/password no passo 4 e a criar a linha `domain.Utilizador` — mas agora só para preencher `utilizador_id` em auditoria, nunca para autenticar; nada nesse passo é validado como credencial de entrada.

**Novo, centralizado em `cmd/admin`:**
- Tabela `administradores` (schema próprio de `migrations_admin/`, sem sqlc — pequena demais para justificar) e `AdministradorRepository`.
- `admin.AuthService`: `PrecisaConfigurar` (true só antes de existir o primeiro administrador), `CriarAdministrador` (rejeita criar um segundo — só pode existir um, de propósito), `Login` com bcrypt.
- `AdminApp` ganhou `administradorAtualID` (campo de instância, 0 = sem sessão) e `exigirSessao()` — todos os bindings que leem/escrevem lojas (`ListarLojas`, `CriarLoja`, `AtualizarLoja`, `AtivarLoja`, `DesativarLoja`, `ObterDashboard`) passam por ali primeiro.
- Frontend: `Configurar.tsx` (primeiro arranque) e `Login.tsx` (arranques seguintes) — a mesma máquina de estados de antes, só que agora em `cmd/admin`; a sidebar do admin ganhou botão "Sair".
- `autoComplete="off"`/`"new-password"` nos campos sensíveis mantido/replicado aqui (ver achado abaixo, válido em ambos os desenhos).

**Achado (válido para os dois desenhos, mantido aqui por ainda se aplicar):** os campos de username/password tinham `autoComplete="username"/"current-password"` — o WebKitGTK deste ambiente de desenvolvimento usa isso (nalguns casos mesmo ignorando o valor) para sugerir dados guardados a nível de sistema (chegou a preencher o username do sistema operativo em vez do que estava a ser escrito). Num terminal POS partilhado, sugerir a sessão de outro operador é um problema real, não só de teste — todos os campos sensíveis passaram a `autoComplete="off"` (passwords novas a `"new-password"`).

**Extensão seguinte, mesmo addendum — `Configurar.tsx` passou a criar a 1ª loja também:** pedido explícito do utilizador: "a app começa com configuração onde seleciona tipo de negócio, nome, logótipo, user e pass do admin — vai para login — abre o Admin Geral sempre". Na primeira volta, `Configurar.tsx` só criava o administrador; o dashboard seguinte ficava vazio até alguém registar uma loja à mão em "Lojas" (caminho de `.db` já configurado por fora). Confirmado por pergunta direta que isto era sobre o `cmd/admin` (não uma reversão do login por loja) — passou a criar a loja e o administrador no mesmo ecrã:
- `ConfigService.FinalizarWizard`/`EstadoConfig` ganharam `Logo` (data URL opcional) gravado em `configuracoes` junto ao tema — dado de negócio, vive dentro da base da própria loja, nunca copiado para `admin.db`.
- `config.ResolveNovaLojaDBPath(nome)`: gera um `.db` novo por loja criada a partir do `cmd/admin`, numa subpasta `lojas/` ao lado de `admin.db` — slug do nome, desambiguado com sufixo numérico se já existir.
- `LojasService.CriarComWizard`: cria o ficheiro, corre o mesmo `FinalizarWizard` que o `cmd/app` correria manualmente (armazém principal fixo em "Principal", utilizador semente da loja preenchido com o nome/username do administrador central — só para atribuição em auditoria, nunca login), só depois regista em `admin.db`.
- `AdminApp.ConfigurarPrimeiroArranque`: administrador primeiro (idempotente — uma segunda tentativa não esbarra em "já existe" se só a loja falhou), loja depois; se a loja falhar, `AvisoLoja` vem preenchido mas a sessão já fica ativa — não bloqueia a entrada, a loja fica por adicionar manualmente depois.
- `Configurar.tsx`: um ecrã só, alinhado ao topo com scroll (`.login-shell--alto`, em vez de centrado — o ecrã ficou alto demais para centrar sem cortar o topo) — nome+tipo de negócio (mesmos presets do wizard do `cmd/app`, via novo binding `ObterPresetsNegocio` no `AdminApp`), logótipo opcional (upload + preview, limite de 1.5MB client-side), depois utilizador/password do administrador.

Testado: 5 testes novos (`ResolveNovaLojaDBPath` — slug, colisão, fallback; `CriarComWizard` válido e sem tipo; `ConfigurarPrimeiroArranque` feliz e com loja a falhar sem perder o administrador). Confirmado visualmente por captura de ecrã real (formulário completo, scroll alinhado ao topo) e ponta a ponta por script Go direto contra um `admin.db` limpo (administrador criado, loja criada com tipo/tema/logo corretos, sessão ativa sem precisar de login extra, `ListarLojas` já devolve a loja).

Testado: `admin.AuthService` com 6 testes (configurar antes de existir administrador, rejeitar segundo administrador, validar campos, login certo, password errada, login antes de existir administrador). Confirmado ponta a ponta via script Go direto contra `admin.db` real (bootstrap → login errado rejeitado → acesso sem sessão bloqueado → login certo → acesso liberado) — não por automação de cliques: o mesmo WebKitGTK do achado acima também substitui texto digitado por `xdotool` por sugestões nativas, tornando essa via pouco fiável neste ambiente; screenshots reais (`xwd`/`netpbm`) confirmaram visualmente os ecrãs de `Configurar`/`Login`/`cmd/app` sem login. Base de dev reconfigurada de farmácia para ferragem (tipo `ferragem`, tema/cor e produtos de exemplo próprios) diretamente por script, sem passar pelo wizard na GUI, pelo mesmo motivo.

---

## 8.7. Addendum (pós-Fase 10) — Categorias: registo, edição, (des)ativação; cmd/admin volta a servir uma só loja

Três pedidos explícitos seguidos, o último a reverter parte do desenho do segundo:

**1. "Implemente o registo de categorias no admin"** — `Categorias` era um item de sidebar do `cmd/admin` sem ecrã nenhum (fidelidade visual ao mock `1c`, sem gestão). Ganhou CRUD real: `admin.CategoriasService` resolve a loja a gerir, abre a ligação (`AbrirLoja`) e delega no mesmo `CategoriaService`/`CategoriaRepository` que o `cmd/app` já usava — categorias continuam dado de negócio, nunca copiadas para `admin.db`.

**2. "Não tem como alterar ou inativar categoria depois de salvar"** — categorias só tinham `Listar`/`Criar`; as outras tabelas de cadastro (armazéns, produtos, clientes) já tinham `ativo` desde a migration inicial (000001), categorias ficou por fazer e nunca teve `Atualizar`. Migration 000007 (`ALTER TABLE categorias ADD COLUMN ativo`, default 1) + `AtualizarCategoria`/`DefinirCategoriaAtiva` no sqlc, no `CategoriaRepository`/`CategoriaService`/`api.App` (mesmo padrão do `ArmazemRepository`/`ArmazemService`) e nos bindings do `admin.CategoriasService`/`AdminApp`. Frontend (`CategoriasSection` em `cmd/app`, `Categorias.tsx` em `cmd/admin`): coluna "Ativo" + botões Editar/Ativar/Desativar — "Editar" reaproveita o formulário de criar (`key={categoria?.ID ?? "nova"}` força remonte ao trocar de categoria em edição, mesmo truque já usado em `Lojas.tsx`).

**3. "Remove a criação de loja no admin porque esse deve funcionar apenas para uma [loja], com que pode ter armazéns. [...] o admin deve ter a cor de tipo de loja setado"** — reverte o desenho multi-loja da Fase 9 (ver secção 10 abaixo): o `cmd/admin` passa a servir sempre a mesma e única loja, criada uma vez em "Configurar" (addendum 8.6, extensão), nunca trocada depois. Confirmado por pergunta direta que o alcance incluía remover "Lojas" da sidebar por completo e simplificar o Dashboard (não só tirar o botão de criar):
- `repository.LojaRepository` perdeu `Listar`/`ListarAtivas`/`Obter(id)`/`Atualizar`/`DefinirAtiva` (só serviam o CRUD multi-loja removido); ganhou `ObterUnica` — a tabela nunca tem mais do que uma linha.
- `admin.LojasService` perdeu tudo exceto `CriarComWizard` (a única forma de criar a loja, sempre a partir de "Configurar"); ganhou `ObterAtual` (nome/tipo/tema da loja).
- `admin.CategoriasService`/`AdminApp`: os bindings de categoria deixaram de receber `lojaID` — resolvem sempre a loja única internamente.
- `admin.DashboardService`: reescrito de agregação multi-loja (cartões por loja, série por tipo de negócio agrupada, "% do grupo") para os KPIs/alertas/série de 14 dias de uma loja só; `Dashboard.TemLoja` cobre o estado "ainda sem loja configurada" (ex: se `CriarComWizard` falhou no primeiro arranque e ninguém a registou manualmente entretanto — já não há "Lojas" para o fazer).
- Cor da app pelo tipo de negócio: `AdminApp.ObterLojaAtual()` novo, chamado uma vez ao entrar em `AdminPrincipal`; `data-tipo-negocio` no `<html>`, `--accent`/`--accent-hover`/`--accent-bg` remapeados por tipo em `App.css`, reaproveitando as cores já auditadas (WCAG, Fase 10) em vez de inventar tons novos — "personalizado" fica no azul neutro genérico. A sidebar mostra o nome real da loja + tag do tipo em vez de "Administração" genérico.

Testado: suites de `lojas`/`categorias`/`dashboard` em `backend/admin` reescritas para o novo desenho (`TemLoja=false` sem loja, loja offline devolve erro em vez de silenciar) — 21 testes a passar; `CategoriaService` em `backend/usecase` ganhou 4 testes (`Atualizar` exige nome e renomeia, `Ativar`/`Desativar`). Confirmado ponta a ponta na base de dev real (Ferragem Cossa): login, sidebar e nav ativa já na cor laranja do tema, Categorias a carregar direto sem seletor de loja nenhum, "Editar"/"Desativar" a funcionar com dados reais — via `xdotool type` diretamente sobre o campo já focado (sem clicar antes num campo por coordenadas, que continua pouco fiável neste WebKitGTK).

**Extensão seguinte, mesmo addendum — ecrã de Login também com a cor do tipo de negócio:** pedido explícito. A cor só se aplicava depois de autenticar (`AdminPrincipal` buscava a loja no seu próprio efeito); `AdminApp.ObterLojaAtual()` deixou de exigir sessão (nome/tipo de negócio não é dado sensível, mesmo critério de `ObterPresetsNegocio`), e `App.tsx` passou a carregar a loja uma vez no topo, antes de decidir qual ecrã mostrar, aplicando `data-tipo-negocio` mesmo antes do Login. `Login.tsx` ganhou o mesmo nome+tag que a sidebar já mostrava. Testado: 1 teste novo confirma `ObterLojaAtual` a funcionar sem sessão (depois de `Logout()`) — 22/22 em `backend/admin`. Confirmado visualmente: ecrã de Login já com logótipo/nome/tag e botão "Entrar" na cor do tema, antes de qualquer autenticação.

---

## 8.8. Addendum (pós-Fase 10) — CRUD completo de produtos, com variações/grupos

Pedido explícito: "crud completo de criação de produtos... sem esquecer que produto pode ter variações ou grupos". Produtos só tinham `Criar` (uma única unidade fixa, sem forma de acrescentar mais depois) e `Ativar`/`Desativar` — sem `Editar` nenhum, sem `Duplicar` exposto em UI nenhuma (o binding já existia desde cedo, nunca teve botão), e sem gerir as "variações/grupos" do mesmo produto (`UnidadeProduto`, SDD 4.1: unidade/caixa, garrafa/dose) além do momento da criação. O SDD já resolvia "variantes" via "Duplicar produto" (ex: mesma cerveja, tamanhos diferentes) — não um modelo de produto-pai/variante à parte.

- `usecase.ProdutoService.AdicionarUnidade`: `CriarUnidade` já existia no `ProdutoRepository`, mas sem usecase nem binding nenhum por cima — era impossível acrescentar uma unidade a um produto já criado, só na criação inicial.
- `api.App.AdicionarUnidadeProduto` novo; `DuplicarProduto` já existia mas nunca tinha UI a chamá-lo em nenhuma das duas apps.
- `admin.ProdutosService` (novo, mesmo padrão de `CategoriasService`): `ListarTodosProdutos`/`ListarUnidadesProduto`/`CriarProduto`/`AtualizarProduto`/`Ativar`/`DesativarProduto`/`DuplicarProduto`/`AdicionarUnidadeProduto`, todos pela única loja. `abrirLojaUnica()` extraído para `loja_conexao.go` nesta altura — `CategoriasService` e `LojasService.ObterAtual` já tinham a mesma lógica copiada duas vezes, `ProdutosService` seria a terceira.
- Frontend: "Produtos" deixou de ser stub no `cmd/admin` (`Produtos.tsx` novo) — criar (nome/categoria/tipo/unidade base/stock mínimo/lote controlado + bloco repetível de variações/grupos, pelo menos uma obrigatória), editar (tipo e unidade base ficam bloqueados no formulário — o próprio `AtualizarProduto` não os aceita mudar), duplicar, ativar/desativar, "+ adicionar unidade" a um produto existente. `ProdutosSection` em `cmd/app` ganhou exatamente o mesmo — e passou a ter a sua própria lista (`ListarTodosProdutos`, incluindo inativos, para poder reativar) em vez do estado `produtos` partilhado com Stock/Inventário/Fichas técnicas/Farmácia, que tem de continuar só com produtos ativos (não faz sentido dar entrada de stock ou usar como ingrediente um produto desativado).

Testado: 9 testes novos (usecase: criar com várias unidades, exige pelo menos uma, `AdicionarUnidade` valida campos, `Duplicar` copia as próprias unidades da cópia, `Atualizar` renomeia; admin: os mesmos fluxos por loja, rejeita sem loja configurada) — 46/46 a passar em `backend/admin`+`backend/usecase`. Confirmado visualmente com dados reais em ambas as apps: tabela com Editar/Duplicar/Desativar, formulário de criar com o bloco de variações/grupos visível, edição a pré-preencher com tipo/unidade base corretamente bloqueados.

---

## 8.9. Addendum (pós-Fase 10) — cmd/app deixa de abrir sozinho, só a partir do botão "PDV"

Pedido explícito: "quero que cmd/app não abra independente mas sim a partir do admin principal dependendo do tipo de loja... remove o botão e funcionalidades de administração em Vendas". Duas opções foram postas ao utilizador antes de avançar — (A) `cmd/admin` lança `cmd/app` como processo próprio, maximizado; (B) fundir a UI de venda para dentro do `cmd/admin`, num único processo. Escolhida a opção A: reaproveita 100% da UI de venda já existente (`VendaDireta.tsx`/`Bottlestore.tsx`, ~900 linhas) sem duplicar nada entre os dois frontends, ao custo de continuarem a ser dois processos/janelas em vez de um só. Antes de avançar, o utilizador perguntou se dava para os dois usarem sempre a mesma base — respondido e implementado (ver abaixo).

- `cmd/app/main.go` ganhou a flag `-db <caminho>`: sem ela, resolve o caminho local da máquina como sempre (`make run-app` continua a funcionar sozinho em desenvolvimento); com ela, abre exatamente esse ficheiro. Sem isto, o `cmd/app` lançado pelo `cmd/admin` abriria uma base **diferente e vazia**, porque a loja criada via `ConfigurarPrimeiroArranque` (addendum 8.6) vive em `~/.config/gestao-retalho-admin/lojas/<slug>.db`, não no caminho por omissão do `cmd/app` (`~/.config/gestao-retalho/loja.db`). Janela passou também a abrir maximizada (`WindowStartState: options.Maximised`) — cobre o ecrã, como pedido.
- `config.ResolveAppExecutavel()` localiza o binário do `cmd/app` para o lançar: `GESTAO_RETALHO_APP_BIN` (override explícito) → ao lado do próprio executável do `cmd/admin` (layout esperado de uma instalação real, os dois lado a lado) → pasta irmã `cmd/app/build/bin` dentro do repositório (só serve em desenvolvimento, onde `make build-app`/`build-admin` deixam os binários em pastas separadas).
- `admin.PDVService.Abrir`: resolve a única loja (`LojaRepository.ObterUnica`), lança `exec.Command(appExe, "-db", loja.CaminhoDB)`; um mutex com estado `aberto` evita abrir uma segunda janela se "PDV" for clicado outra vez enquanto a primeira ainda está a correr (reposto a `false` quando o processo termina, via `cmd.Wait()` numa goroutine). `AdminApp.AbrirPDV` (binding novo) nunca renderiza nada dentro do `cmd/admin` — só arranca o processo.
- Frontend: botão "PDV" novo na sidebar do `cmd/admin`, destacado com a cor de acento (ação, não navegação — não faz parte da máquina de estados `Ecra`). `AppPrincipal` em `cmd/app` perdeu o estado `modo`, o switch "Venda/Administração" e o import de `Administracao` — mostra sempre o ecrã de venda certo consoante os módulos ativos (ou um aviso se a loja não tiver nenhum módulo de venda ativo).
- **Consequência assumida conscientemente:** `Administracao.tsx` e os ecrãs que só lá viviam (Armazéns, Stock, Inventário, Fichas técnicas, Farmácia, Clientes e dívidas, Relatórios) ficam no repositório mas **inacessíveis por agora** — só Produtos e Categorias já tinham sido portados para o `cmd/admin` (addendums 8.7/8.8) antes desta mudança. Os restantes continuam por portar; até lá, essa gestão não tem ecrã nenhum ligado em nenhuma das duas apps.

Testado: 5 testes novos (`ResolveAppExecutavel` com override de ambiente e sem binário nenhum presente; `PDVService` rejeita sem loja configurada, propaga erro do resolvedor de executável, lança o processo com o caminho certo) — 51/51 a passar. Confirmado ponta a ponta: login no `cmd/admin` (Ferragem Cossa) → clique em "PDV" → `cmd/app` abre maximizado (3840x2123, cobre o ecrã inteiro no ambiente de teste) com os mesmos produtos da mesma base de dados, sem nenhum botão de Administração visível.

---

## 10. Fase 9 — Admin universal multi-loja — tela `1c`

> **Nota (addendum 8.7):** o desenho multi-loja desta fase — registo de várias lojas, dashboard agregado com cartões por loja e comparação entre negócios — foi revertido a pedido explícito do utilizador depois da Fase 10. Esta secção fica como registo histórico da implementação original; o estado atual está descrito no addendum 8.7 acima.

- `cmd/admin/main.go`: mesmo `CoreModule`, mas injeta lista de `DBPath` (uma por loja) em vez de uma única conexão (secção 8.5 do SDD).
- Repositórios/serviços do admin operam sobre múltiplas conexões SQLite simultâneas, agregando nos relatórios comparativos.
- Implementação fiel da tela `1c`: sidebar de navegação (Dashboard, Produtos, Fichas técnicas, Armazéns, Stock, Inventário, Vendas, Clientes e dívidas, Relatórios, Utilizadores, Lojas), KPIs do grupo no topo (vendas hoje, lucro estimado, contas abertas agora, dívidas em aberto), gráfico de barras empilhadas "vendas por loja últimos 14 dias", cartões por loja com tag de tipo de negócio colorida (dourado bottlestore / laranja ferragem / verde mercearia / teal farmácia, conforme secção 7.3), painel de alertas (stock mínimo, dívidas antigas, contas abertas há muito tempo, desvio consumo teórico vs real), painel "mais vendidos hoje".
- CRUD de "Lojas" (gestão de quais `.db` o admin conhece).
- Nota de evolução do SDD (8.5): não implementar transporte HTTP agora — só desenhar o `CoreModule` de forma desacoplada o suficiente para isso ser incremental no futuro, se lojas ficarem fisicamente separadas.

> **Estado: concluída**, com duas exclusões deliberadas. Decisões tomadas durante a implementação:
> - O snippet do SDD 8.5 (`fx.Provide(func() []config.DBPath {...})`) não é literal — `CoreModule` foi desenhado para UMA base (um `*sql.DB`, um `*api.App`), e um `[]DBPath` não encaixa nesse grafo sem reescrever `CoreModule` para N instâncias simultâneas. A solução ficou em `admin.AbrirLoja(caminhoDB string)`: reaproveita `di.CoreModule` **tal e qual**, num container Fx próprio e descartável, uma vez por loja, sempre que o dashboard agrega dados — abre, faz as perguntas que precisa através dos mesmos bindings que o `cmd/app` usaria, fecha a ligação. Nunca mantém N ligações permanentes; SDD 8.3 ("core 100% partilhado") fica literalmente verdade, não só em espírito.
> - Isto criava um ciclo de imports (`di` → `admin` → `di`) se o módulo Fx do admin (`admin.db` + `LojasService` + `DashboardService` + `AdminApp`) vivesse em `backend/di`. Resolvido pondo esse módulo (`admin.Module`) dentro do próprio `backend/admin` — só este pacote importa `di` (para `CoreModule` via `AbrirLoja`), nunca o inverso. `AdminApp` também vive em `admin`, não em `api`, pela mesma razão (evitar `api` → `admin` → `api`).
> - `admin.db` (registo de lojas) tem schema e migrations próprios (`migrations_admin/`, embed separado) — nunca reaproveita o schema completo de uma loja só para guardar `id, nome, caminho_db, ativo`; é uma tabela sem sqlc (schema pequeno demais para justificar gerar código à parte).
> - `LojasService.Criar` liga-se de facto à base antes de aceitar o registo (via `AbrirLoja` + `EstaConfigurado()`) — rejeita caminho inexistente e loja que ainda não terminou o wizard, para o erro aparecer no momento de adicionar a loja, não mais tarde como um cartão "offline" sem contexto no dashboard.
> - O gráfico "vendas por loja, últimos 14 dias" agrupa por **tipo de negócio**, não por loja individual — bate com a legenda do mock (dourado/laranja/verde/teal são tipos, não lojas); duas lojas do mesmo tipo somam no mesmo segmento. "Mais vendidos hoje" seguiu a mesma convenção de cor.
> - Limiares dos alertas sem número fixo no SDD, escolhidos por bom senso: conta aberta há mais de 4h (cobre um turno de bar sem soar em contas normais de uma noite), desvio de consumo teórico vs real acima de 10% (relatório da Fase 8, janela de 30 dias — a mesma da dívida antiga), dívida com mais de 30 dias (já definido na Fase 7).
> - O relatório "Comparativo entre negócios" (SDD 6), deixado propositadamente por implementar na Fase 8 por não haver o que comparar dentro de uma loja, fica **satisfeito por este próprio dashboard** — não há um ecrã de relatório à parte para isso.
> - **Exclusão 1:** só "Dashboard" e "Lojas" têm ecrã funcional — o resto da sidebar do mock (Produtos, Fichas técnicas, Armazéns, Stock, Inventário, Vendas, Clientes e dívidas, Relatórios, Utilizadores) existe por fidelidade visual, mas mostra uma nota a dizer para usar o executável da loja; gestão detalhada por loja continua ali, o admin foca-se na visão agregada (SDD 8.5: "visão agregada primeiro").
> - **Exclusão 2:** sem seletor de ficheiro nativo em "Adicionar loja" — o caminho do `.db` é um campo de texto. Não há diálogo de ficheiro Wails testável neste ambiente sem automação de cliques; um caminho escrito à mão é funcionalmente equivalente e não bloqueia nada.
> - Testado com uma loja real (a `loja.db` de desenvolvimento das fases anteriores, farmácia já configurada): registo via `AdminApp.CriarLoja`, seed de um produto/stock/venda através do `cmd/app` normal, e `ObterDashboard()` a devolver KPIs corretos (vendas, lucro, margem, série de 14 dias, mais vendidos) — confirmado com `cmd/app` e `cmd/admin` a correr ao mesmo tempo sobre o mesmo ficheiro SQLite sem erros.

---

## 11. Fase 10 — Polimento visual e testes com utilizadores reais

- Auditoria visual das 5 telas do design vs. implementação real (pixel-adjacent, não pixel-perfect — grelha 4px/8px e tokens de cor devem bater certo).
- Ícones: outline fino para bottlestore, sólido para ferragem, biblioteca única neutra (Lucide/Feather) para admin — conforme secção 7.
- Testes de usabilidade com operadores reais de balcão/bar (a app tem alvo explícito de "< 200ms percecionados" em adicionar item / fechar conta — validar na prática, não só em teoria).
- Revisão de acessibilidade básica de contraste nos 4 temas (bottlestore em fundo escuro é o mais arriscado).

> **Estado: concluída**, com uma exclusão que não é possível resolver a partir do código. Decisões e achados:
> - **Auditoria de contraste (WCAG 1.4.3), feita a sério, não por amostragem:** um script calculou o rácio de contraste real (fórmula de luminância relativa da WCAG) de cada par texto/fundo dos 5 temas. Resultado surpreendente — **não foi o bottlestore (fundo escuro) que falhou**, apesar de o SDD o apontar como "o mais arriscado". Todos os pares do bottlestore passam com folga (o mais apertado, `text-muted` sobre superfície, 4.61:1). Quem falhava eram os **quatro temas de fundo claro**:
>   - O botão principal "Finalizar venda" (fundo = accent, texto branco) só tinha 3.4:1 (ferragem), 4.0:1 (mercearia) e 3.5:1 (farmácia) — abaixo do mínimo de 4.5:1 para texto normal. Corrigido escurecendo `--color-accent` ~18% nesses três temas (ferragem `#E8611C`→`#BE4F16`, mercearia `#3E8E5A`→`#327449`, farmácia `#3A8FD4`→`#2F75AD`) — continuam inequivocamente laranja/verde/azul, só deixaram de falhar (4.9/5.6/4.9:1).
>   - `--color-warning` (`#E0A62B`, dourado do bottlestore, copiado sem alterar para os outros 4 temas) só atinge 8.5:1 em fundo escuro; em fundo claro cai para ~2.0:1, bem abaixo do mínimo de 3:1 até para ícone/texto grande. Corrigido para `#916B1B` nos quatro temas claros (4.3–4.9:1); bottlestore ficou intocado.
>   - `--color-text-muted` do ferragem tinha 4.32:1 (por 0.18 abaixo do mínimo) — escurecido para `#636A77` (4.86:1).
>   - Nenhuma mudança de fundo/texto principal foi necessária — esses já passavam todos com grande margem (13–17:1).
> - **Ícones:** adicionada `@heroicons/react` (uma dependência só, cobre outline e sólido pelo mesmo pacote — mais simples do que combinar Lucide + uma biblioteca sólida à parte só para a ferragem). Aplicados onde têm valor real, não em cada botão do ecrã: sidebar + KPIs do `cmd/admin` (outline neutro, SDD 7.3), ações principais do `Bottlestore.tsx` — nova conta / voltar / cancelar / fechar conta (outline, SDD 7.1), e do `VendaDireta.tsx` — busca / carrinho / finalizar (SDD 7.2: sólido só quando `tipoNegocio === "ferragem"`, outline para mercearia/farmácia, decidido em runtime porque os três tipos partilham o mesmo ecrã). Passo do wizard concluído trocou o caráter "✓" por um `CheckIcon` real.
> - **Auditoria visual/grelha de espaçamento:** o SDD 7.4 sugere uma grelha de 4px/8px, mas **o próprio mock aprovado não a segue** — os valores inline do mock usam uma escala mais fina (9px, 11px, 13px, 14px, 17px, 18px, 22px, 33px...). Confirmado por comparação direta ecrã a ecrã (ex: `2c`/ferragem: botão "Finalizar venda" no mock é `padding:16px`, gap `10px`, título `15px`; o `VendaDireta.css` já implementado usa os mesmos números, herdados de quando os ecrãs foram construídos nas Fases 3/4). Corrigir para uma grelha estrita teria **afastado** a implementação do mock em vez de a aproximar — por isso não foi feito; o que este passo confirmou é que a implementação já bate com os valores reais do mock, não com uma regra abstrata que nem o próprio mock segue.
> - **Achado de performance, fora do que foi pedido mas dentro do espírito do "< 200ms percecionados":** `Bottlestore.tsx.adicionar()` fazia duas viagens sequenciais ao backend por clique — `AdicionarItemConta` e depois `ListarItensConta` completo só para reconfirmar o que acabou de criar. `AdicionarItemConta` já devolve o item criado; passou a ser inserido localmente (1 viagem em vez de 2, na ação mais repetida do ecrã, uma noite inteira de bar). `transferir()` teve o mesmo tratamento (remove o item localmente em vez de reler a lista toda). `VendaDireta.tsx` já não tinha este problema — "adicionar ao carrinho" já era só estado local, sem chamada nenhuma ao backend até "Finalizar venda".
> - **Exclusão, genuinamente fora do alcance de um agente:** "Testes de usabilidade com operadores reais de balcão/bar" exige pessoas reais a usar a app fisicamente — não é algo que se possa executar a partir do código. Fica como checklist para a equipa correr:
>   1. Cronometrar (relógio, não "sensação") 10 repetições de "adicionar item ao carrinho → item aparece" e 10 de "fechar conta → recibo pronto", em hardware real de balcão, não numa VM de dev.
>   2. Um operador que nunca viu a app tenta completar uma venda de 3 itens com desconto sem ajuda — cronometrar e anotar onde hesita.
>   3. Testar com uma mão a segurar copo/produto e a outra no rato/teclado (cenário real de bar) — algum alvo de clique é pequeno demais?
>   4. Repetir os passos 1–3 em bottlestore (pouca luz, tema escuro) e ferragem/mercearia/farmácia (luz de loja, tema claro).
>   5. Deixar o operador tentar cancelar uma conta e reverter um item por engano — os textos de confirmação fazem sentido para quem não leu o SDD?

---

## 12. Requisitos transversais (aplicar desde a Fase 1, não deixar para o fim)

- **Offline-first:** nunca introduzir dependência de rede no caminho crítico de venda.
- **Auditoria:** todo `movimento_stock` e `venda` grava `utilizador_id` + timestamp; nunca apagar, usar `cancelada`/`estornada`.
- **Performance:** operações de POS devem responder em < 200ms percecionados — medir desde a Fase 3, não só na Fase 10.
- **Backup:** rotina de cópia do `.db` (diária) — implementar já na Fase 1, é barato e crítico.
- **Impressão térmica (ESC/POS):** deixado como extensão futura explícita no SDD — não bloqueia nenhuma fase, mas os fluxos de "imprimir recibo" devem já estar desenhados como um passo opcional/plugável desde a Fase 3.

---

## 13. Empacotamento e distribuição

- `wails build` para os alvos necessários (Windows/Linux, e macOS se aplicável) a partir da Fase 0 (build limpo é parte da definição de "pronto" de cada fase).
- Um único instalador por SO — sem builds separados por tipo de negócio (reforça a decisão arquitetural central do SDD, secção 9.7).
- `cmd/admin` empacotado separadamente, mesmo core partilhado.

---

## 14. Sequenciamento sugerido

Ordem estritamente sequencial (cada fase depende da anterior tecnicamente, exceto onde indicado):

```
Fase 0 (scaffolding)
   → Fase 1 (core stock/produtos)
      → Fase 2 (wizard/config)
         → Fase 3 (PDV direto)          → Fase 6 (nicho: farmácia/mercearia, pode correr em paralelo com Fase 4/5)
            → Fase 4 (bottlestore contas)
               → Fase 5 (fichas técnicas)
                  → Fase 7 (clientes/dívidas, pode começar em paralelo com Fase 4)
                     → Fase 8 (relatórios/exportação)
                        → Fase 9 (admin universal)
                           → Fase 10 (polimento)
```

A Fase 6 (farmácia/mercearia) e a Fase 7 (clientes/dívidas) são as mais fáceis de paralelizar com outra pessoa da equipa, porque tocam pouco código partilhado com bottlestore/fichas técnicas.

---

## 15. Perguntas em aberto para confirmar contigo

1. As decisões de arquitetura da secção 0 (sqlc, driver SQLite puro-Go, sem Tailwind, sem Redux) estão de acordo, ou preferes outra combinação?
2. Existe prazo/MVP-alvo específico (ex: "ferragem a vender até X data") que deva reordenar as fases acima?
3. O admin universal (Fase 9) vai correr sempre na mesma máquina/rede local das lojas, como assume o SDD, ou já sabes que vai precisar de acesso remoto (o que traria a camada HTTP mencionada como evolução futura na secção 8.5 mais cedo)?
