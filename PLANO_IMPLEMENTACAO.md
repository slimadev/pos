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
> - Frontend da Fase 1 (`cmd/app/frontend/src/FaseUm.tsx`) é deliberadamente sem tema/estilo — existe para exercitar os bindings ponta a ponta, não para o operador final usar; é substituído pelos ecrãs `1b`/`2a`/`2b`/`2c` nas fases seguintes.

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

---

## 5. Fase 4 — Bottlestore: contas abertas — telas `2a` e `2b`

- `usecase.ContaService`: abrir conta (identificador livre + cliente opcional), lançar consumo incremental, listar contas abertas em tempo real, transferir itens entre contas, cancelar conta com justificação, fechar conta.
- Tela `2a`: grelha de cartões de conta com KPIs no topo (nº de contas abertas, total em consumo), filtros "Todas / Com cliente / Abertas > 2h", indicador de tempo por conta com cor (normal vs alerta > 2h, replicando os tons `corTempo`/`fundoTempo` do mock).
- Tela `2b`: grelha de produtos por categoria à esquerda + comanda viva à direita + modal de fecho. O modal de fecho é o componente de pagamento **partilhado com a Fase 3** (split de métodos, cálculo de "falta atribuir" em tempo real, opção de dívida só ativa se cliente identificado — regra 4.10/4.11).
- `dividas`: ao fechar com dívida, cria registo em `dividas`; pagamento de dívida é fluxo independente de nova venda (4.11).
- Regra de bloqueio: dívida só permitida com cliente não-anónimo — validar tanto no frontend (desabilitar opção) como no backend (não confiar só na UI).

**Saída:** bottlestore consegue abrir várias contas em paralelo, lançar consumo ao longo da noite, transferir itens entre contas, e fechar com pagamento misto ou dívida.

**Testes:** conta não pode fechar duas vezes; transferência de item entre contas mantém total consistente nas duas; dívida bloqueada para cliente anónimo.

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

---

## 7. Fase 6 — Módulos de nicho (farmácia e mercearia)

- **Farmácia:** tabela `lotes_produto`, lógica FEFO (vender primeiro o que expira mais cedo) na baixa de stock, alertas de validade próxima, tabela `receitas_medicas` associada à venda (médico, nº de receita, validado por).
- **Mercearia — venda a granel:** sem tabela nova (reutiliza `unidades_produto` com `unidade_base = kg`); UI precisa aceitar input de peso livre (ex: 0.350 kg) na grelha de venda. Deixar ponto de extensão comentado no código para integração futura com balança eletrónica via porta série/USB — não implementar agora, só não bloquear a interface para isso.
- Ambos os módulos só aparecem no menu/ecrãs se ativos em `configuracoes` (mesma mecânica da Fase 2).

**Saída:** farmácia vende com controlo de lote/validade e receita médica; mercearia vende produtos a peso livre.

---

## 8. Fase 7 — Clientes e dívidas completos

- CRUD completo de clientes, histórico de compras e dívidas por cliente.
- Bloqueio opcional (alerta, não obrigatório) de nova dívida acima do `limite_credito`.
- Alertas de dívidas antigas (> 30 dias) — reutilizados depois no painel "Alertas" do dashboard admin (`1c`).
- Pagamento parcial/total de dívida como fluxo independente.

---

## 9. Fase 8 — Relatórios e exportação Excel/PDF

Implementar a tabela completa da secção 6 do SDD, todos com filtro por período/armazém/negócio e exportação:

- `RelatorioService` com um método por relatório (vendas por período, lucro por período, vendas por produto, consumo teórico vs real de ingredientes, vendas por método de pagamento, stock atual, movimentos de stock, perdas e consumo interno, contas abertas/fechadas, dívidas em aberto, fecho de caixa diário, comparativo entre negócios).
- `ExportService`: `excelize` para `.xlsx` (folhas separadas, cabeçalho formatado, totais a negrito), e para PDF avaliar entre `gofpdf` (tabulares simples) vs `chromedp`/`go-wkhtmltopdf` (recibos e relatórios com gráfico) — decisão a confirmar quando chegarmos aqui, conforme complexidade real dos relatórios com gráfico.
- Botão "Exportar" consistente em todos os ecrãs de relatório (padrão já visível no header da tela `1c`).
- O relatório "Consumo teórico vs real de ingredientes" depende da Fase 5 (fichas técnicas) e da Fase 1 (inventário) já estarem implementadas — é o relatório que fecha o ciclo de reconciliação mencionado na secção 3.3.

---

## 10. Fase 9 — Admin universal multi-loja — tela `1c`

- `cmd/admin/main.go`: mesmo `CoreModule`, mas injeta lista de `DBPath` (uma por loja) em vez de uma única conexão (secção 8.5 do SDD).
- Repositórios/serviços do admin operam sobre múltiplas conexões SQLite simultâneas, agregando nos relatórios comparativos.
- Implementação fiel da tela `1c`: sidebar de navegação (Dashboard, Produtos, Fichas técnicas, Armazéns, Stock, Inventário, Vendas, Clientes e dívidas, Relatórios, Utilizadores, Lojas), KPIs do grupo no topo (vendas hoje, lucro estimado, contas abertas agora, dívidas em aberto), gráfico de barras empilhadas "vendas por loja últimos 14 dias", cartões por loja com tag de tipo de negócio colorida (dourado bottlestore / laranja ferragem / verde mercearia / teal farmácia, conforme secção 7.3), painel de alertas (stock mínimo, dívidas antigas, contas abertas há muito tempo, desvio consumo teórico vs real), painel "mais vendidos hoje".
- CRUD de "Lojas" (gestão de quais `.db` o admin conhece).
- Nota de evolução do SDD (8.5): não implementar transporte HTTP agora — só desenhar o `CoreModule` de forma desacoplada o suficiente para isso ser incremental no futuro, se lojas ficarem fisicamente separadas.

---

## 11. Fase 10 — Polimento visual e testes com utilizadores reais

- Auditoria visual das 5 telas do design vs. implementação real (pixel-adjacent, não pixel-perfect — grelha 4px/8px e tokens de cor devem bater certo).
- Ícones: outline fino para bottlestore, sólido para ferragem, biblioteca única neutra (Lucide/Feather) para admin — conforme secção 7.
- Testes de usabilidade com operadores reais de balcão/bar (a app tem alvo explícito de "< 200ms percecionados" em adicionar item / fechar conta — validar na prática, não só em teoria).
- Revisão de acessibilidade básica de contraste nos 4 temas (bottlestore em fundo escuro é o mais arriscado).

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
