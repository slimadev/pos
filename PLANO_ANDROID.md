# Plano de Implementação — App Android (Admin + PDV) para terminais POS portáteis

> Complementa o `PLANO_IMPLEMENTACAO.md` e o `SDD_Gestao_Bottlestore_Ferragem.md`. Alvo: terminais Android portáteis com leitor de código de barras e impressora térmica de talões incorporados (estilo "handheld POS", ecrã ~5,5–6", papel de 58 mm).

---

## 0. Decisões fixadas

| Decisão | Escolha | Consequência |
|---|---|---|
| Onde vivem os dados | **Base SQLite local no próprio terminal**, 100% offline | O terminal é uma loja autónoma, igual a um PC hoje. Não há sincronização com o PC nesta fase (ver §9) |
| Âmbito | **Admin completo + PDV** na mesma app | A app junta o que hoje são dois executáveis (`cmd/admin` + `cmd/app`) |
| Lógica de negócio | **Reaproveitar o Go existente tal como está** (`backend/…`) | Nada de reescrever regras em Kotlin. O Go corre dentro do APK |
| Interface | **Reaproveitar os dois frontends React** (`cmd/admin/frontend`, `cmd/app/frontend`) | Não se copia código. O mesmo código passa a ser compilado também para Android, com ajustes de layout para ecrã pequeno |
| Pasta no repositório | **`gestao-retalho/cmd/android/`** | Segue a convenção atual: `cmd/` contém um executável por pasta e `backend/` é partilhado |
| Quem usa o terminal | **O terminal abre no PDV** (PIN do operador); a administração fica atrás do botão "Administração", que pede sempre a password | Operadores **obrigatórios**: sem eles o PDV não vende (no backend, em `exigirTurno`) e o admin entra direto em Utilizadores com o aviso "Último passo". A regra vale também no desktop (ver `cmd/android/README.md`) |

### 0.1 Equipamento-alvo: **Sunmi V2 Pro** (já disponível para testes)

| Característica | Valor (foto "About device") | O que implica |
|---|---|---|
| Sistema | SUNMI OS 3.6.5, **Android 7.1.2 (API 25)** ✅ confirmado por adb | `minSdk 25` |
| **WebView** | **`com.google.android.webview` 62.0.3202.73 (Chrome 62, 2017)** ✅ | ⚠️ **Bloqueante**, ver §0.2 |
| Google Play | `com.android.vending` + `com.google.android.gms` instalados ✅ | Dá para atualizar a WebView pela Play Store |
| CPU | Qualcomm MSM8917 (4× Cortex-A53) | Fraca: arranque a frio e listas grandes devem ser medidos no A0 |
| RAM / disco | 1,9 GB / 16 GB (10 GB livres) | Chega para SQLite + Go, mas o React tem de ser leve (sem listas de milhares de linhas sem paginação) |
| Ecrã | 5,99", 720 × 1440 (retrato) | ~360 × 720 px CSS: confirma a Fase A5 como prioritária |
| Impressora | Térmica interna **58 mm** (384 px/linha) | Talão de **32 colunas**. Usar o serviço de impressão da Sunmi (§6.1) |
| Leitor | **Tem cabeça de leitura física** ✅ (`persist.scanner.have=1`, serviço `com.sunmi.scanner` 2.7.3 com `IScanInterface`). Também tem leitor por câmara (`com.sunmi.sunmiqrcodescanner`) | Modo broadcast como principal (§6.2) e câmara como recurso |
| Impressora | Serviço `woyou.aidlservice.jiuiv5` presente ✅ | É o que o `printerlibrary` da Sunmi usa |
| Extras | NFC/contactless (no topo laranja) | Fora de âmbito, mas possível mais tarde |

**ABI:** ✅ `ro.product.cpu.abilist = armeabi-v7a,armeabi`. O Android é **só 32 bits**, apesar do CPU de 64 bits. O APK tem de incluir `armeabi-v7a` (gomobile `-target=android/arm`), e o `modernc.org/sqlite` tem de funcionar em `android/arm`. É o primeiro teste do A0.

**RAM livre medida:** ~1,27 GB disponíveis com o sistema em repouso. Chega.

### 0.2 ⚠️ WebView 62: o problema principal

O Chrome 62 não suporta funcionalidades que os frontends atuais usam:

| Funcionalidade | Chrome mínimo | Uso atual |
|---|---|---|
| `gap` em flexbox | 84 | **137 ocorrências** de `gap`/`inset`/`aspect-ratio`/`:is()` no CSS (75 no admin, 62 no PDV). Sem isto, os espaçamentos desaparecem e o layout fica colado |
| `?.` e `??` no JS | 80 | Resolve-se no build (Vite `target`) |
| `import()` dinâmico | 63 | Resolve-se com `@vitejs/plugin-legacy` |

**Tentativa feita:** atualizar a WebView pela Play Store **não foi possível** neste terminal. Também não há outro browser instalado. Por isso a app tem de funcionar com o **Chrome 62**.

Detalhe do CSS (contado nos ficheiros-fonte): quase tudo é `gap` em flexbox/grid, mais 4× `inset` e 1× `:has()`. Nada de `aspect-ratio`, `:is()` ou `:where()`.

Opções, pela ordem recomendada:

1. **Converter o CSS no build, só no modo android (recomendado).** O código-fonte e o desktop ficam iguais.
   - `gap` em grid → `grid-gap`, que o Chrome 62 suporta (`postcss-gap-properties`).
   - `gap` em flex → margens equivalentes (`flex-gap-polyfill`, PostCSS).
   - `inset` → `top/right/bottom/left` (`postcss-preset-env`).
   - O único `:has()` passa a uma classe definida pelo React.
   - JS: `@vitejs/plugin-legacy` com alvo `chrome 62`.
   - Validar visualmente no terminal, ecrã a ecrã, na Fase A5.
   - **Teste visual feito no V2 Pro (28/10/2026):** APK de demonstração (`cmd/android`, ponte com dados de exemplo).
     - ✅ O React 19 arranca no Chrome 62, a ponte `window.go` funciona e o login fica perfeito.
     - ✅ O dashboard carrega os dados.
     - ⚠️ O `flex-gap-polyfill` (margens negativas) tem defeitos: ícone colado ao texto no menu, botão PDV sobreposto ao Dashboard e barras do gráfico encavalitadas.
     - ✅ **Resolvido:** o polyfill foi substituído por um plugin próprio, `cmd/android/web/postcss-gap-chrome62.js`. Em grid usa `grid-gap`. Em flex põe margem nos filhos: coluna → `* + *` com margem em cima; linha → margem à direita, exceto no último filho (salvo se for um ícone); com `flex-wrap` usa margens com compensação negativa no contentor.
     - ✅ **Fase A5 iniciada (admin):** `src/mobile.css` (≤ 600 px) com barra superior e botão ☰, sidebar em gaveta, ecrãs a uma coluna, tabelas com scroll horizontal, modais em ecrã inteiro, `select` brancos, alvos de toque maiores e compensação do `display: contents`. Verificado no V2 Pro: Login, Dashboard, menu, Stock, Produtos, Vendas e modal "Criar produto". O desktop fica igual.
     - ✅ **PDV no terminal:** o frontend `cmd/app` também compila em modo android (`assets/web/pdv`) e abre numa `PdvActivity` por cima do admin. O "voltar" ou o ‹ fecham-no e regressam ao admin sem novo login.
       - Layout: catálogo em 2 colunas, e o carrinho é um painel em ecrã inteiro aberto por uma barra de resumo no fundo.
       - Verificado: venda completa (produtos → carrinho → "Tudo" → finalizar → venda concluída → recibo de 58 mm).
       - Por fazer: escolha de operador por PIN no layout estreito. O "Imprimir" do recibo depende da Fase A3 (impressora Sunmi).
     - ✅ **Polimento (Android + exe):**
       - Ícones de app novos (`design/icones/`, gerados para os dois exe e para o Android).
       - Ícones Heroicons em ~145 botões dos dois frontends.
       - Tabelas do admin em cartões no ecrã estreito (`tabelasResponsivas.ts`).
       - Bottlestore no telemóvel: comanda fixa em baixo, fecho em folha. Tempo de conta aberta nos cartões, também no desktop.
       - Fontes dos temas embutidas (`@fontsource`, funcionam offline).
       - Demo com 4 lojas (ferragem, mercearia, farmácia, bottlestore) e o leitor de códigos já ligado no catálogo (evento `codigo-lido`).
       - Desktop verificado a 1280×800 no Edge/Chromium (o mesmo motor da WebView2) com os 4 tipos.
     - ✅ **Backend Go real no terminal (Fases A0–A2):**
       - `cmd/android/gobridge` (gomobile, `android/arm`).
       - Ganchos no backend: pasta de dados, abrir ficheiros e seletor de ficheiros. Sem gancho, o comportamento do desktop não muda.
       - Lado Android: `RetalhoApp`, `PonteJs` e `PlataformaAndroid`, com a ponte JS real em `web/ponte.js`.
       - Verificado no V2 Pro: primeiro arranque → loja "Ferragem Central" → login → entrada de stock (100 m a 60 MT) → venda no PDV (3 m, 285 MT) → stock 97 → dashboard e Vendas com a venda real.
       - Sondagem de desempenho: abrir a base 0,03 s; catálogo 0,005 s.
       - **Fuso horário:** o Go no Android ficava em UTC e o SQLite (`date(x,'localtime')`) também, porque a sua libc lê o `TZ` uma só vez e sem base de fusos. Corrigido em `gobridge/fuso*.go`, com teste.
       - **Ambiente:** `scripts/instalar-ambiente.sh`, sem sudo. Os downloads forçam IPv4, porque o IPv6 desta rede deixava-os a ~7 KB/s.
       - `golang.org/x/mobile` fixado numa versão compatível com o Go 1.25 (as mais recentes exigem 1.26).
     - ✅ **Fases A4, A6 e A7 (sem impressão, à espera de papel):**
       - **Cópias de segurança** (`backend/copias`, ecrã *Gestão → Cópias de segurança*): criar, partilhar e "Guardar em…" pelo seletor do Android. O restauro é aplicado no arranque e corrige os caminhos das lojas, por isso uma cópia passa entre terminal e PC. Também pode ser feito no próprio ecrã de primeiro arranque. Há alerta no dashboard. Verificado no V2 Pro: desinstalar → instalar o APK de distribuição → restaurar → loja, stock e vendas de volta.
       - **Leitor de códigos Sunmi:** broadcast passado ao JS como `codigo-lido`, mais o modo teclado na `ponte.js`. Ouvem-no o PDV, a Receção, o Inventário (soma à contagem) e os Produtos.
       - **PIN do operador no telemóvel:** teclado do ecrã grande e sem o teclado do sistema.
       - **APK de distribuição assinado:** versão do `wails.json`, chave fora do repositório e `build-apk.ps1 -Release`.
       - Diálogos `alert`/`confirm`/`prompt` nativos, e rótulos visíveis no formulário de movimento de stock.
2. **Instalar uma WebView/Chrome mais recente por `adb install`** na preparação do terminal. No Android 7 o próprio Chrome pode servir de WebView, a escolher em *Developer options → WebView implementation*. Exige o APK oficial assinado pela Google. Fica como melhoria opcional, não como requisito.
3. **Embutir um motor próprio (GeckoView)**: deixa de depender da WebView do sistema, mas acrescenta ~50 MB e o suporte a Android 7 nas versões recentes é incerto.
4. **Interface nativa (Kotlin/Compose) só para os ecrãs do terminal**: a melhor experiência, mas obriga a reescrever a interface (semanas), embora a lógica Go se mantenha.

**Ambiente de desenvolvimento já instalado no Windows:** Android SDK, NDK 28.2, `adb`, JDK 21. Falta só o `gomobile`.

---

## 1. Arquitetura

O desktop funciona assim hoje: **React → bindings do Wails (`window.go.*`) → structs Go (`AdminApp`, `api.App`) → usecases → SQLite.**

Em Android trocamos só a camada do meio, o transporte que o Wails fornece, por uma ponte própria:

```
┌──────────────────────── APK ─────────────────────────────────────────┐
│  Kotlin (Activity)                                                   │
│   ├─ WebView ── serve assets/web/{admin,pdv} via WebViewAssetLoader  │
│   │     └─ ponte-wails.js  → define window.go.admin.AdminApp.*       │
│   │                           e window.go.api.App.* (igual ao Wails) │
│   ├─ JavascriptInterface "Ponte" ──► gobridge.Chamar(alvo, método,   │
│   │                                   argsJSON) → resultadoJSON      │
│   ├─ Impressora  (SDK do fabricante | ESC/POS | imagem)              │
│   └─ Leitor      (teclado | broadcast intent | câmara)               │
│                                                                      │
│  libgobridge.so  (gomobile bind)                                     │
│   └─ cmd/android/gobridge ─► admin.Module + di.CoreModule (Fx)       │
│                              ─► backend/usecase ─► loja.db (SQLite)  │
└──────────────────────────────────────────────────────────────────────┘
```

Esta abordagem funciona por três motivos:

- **Os frontends só falam com o Go através de `wailsjs/go/*`.** Já verifiquei que não usam nenhuma função do runtime do Wails. Cada função gerada faz só `window['go']['admin']['AdminApp']['Metodo'](...)`. Se `window.go` tiver a mesma forma, o React funciona sem alterações.
- **O Wails já usa `encoding/json` para transportar argumentos e resultados.** A ponte faz o mesmo, por isso datas, structs e erros chegam com o formato que os tipos de `models.ts` já esperam.
- **O SQLite usa `modernc.org/sqlite`, que é Go puro e não precisa de CGO.** Isto facilita a compilação para ARM. Validar no spike (Fase A0).

### 1.1 Alternativas consideradas e rejeitadas

| Alternativa | Porque não |
|---|---|
| Reescrever em Kotlin/Flutter | Duplica toda a lógica (stock, custo médio, fichas técnicas, faturação…) e os testes existentes. É exatamente o que queremos evitar |
| Wails v3 com suporte mobile | O suporte Android ainda não é estável. Arriscado para um produto em terminais de clientes |
| Android como cliente HTTP do PC | Descartado por decisão (§0): sem rede a loja parava |
| Capacitor/Ionic + Go | Acrescenta mais uma camada, com o mesmo resultado que WebView + gomobile |

---

## 2. Estrutura de pastas nova

```
gestao-retalho/
  backend/                      ← sem mudanças de estrutura (só os ajustes do §3)
  cmd/
    app/  admin/                ← iguais; frontends ganham modo "android" (§5)
    android/
      README.md                 ← como compilar e instalar no terminal
      gobridge/                 ← pacote Go exportado com gomobile bind
        bridge.go               ← Iniciar(dirDados, plataforma), Chamar(alvo, metodo, argsJSON)
        despacho.go             ← reflexão: método por nome → decode args → call → encode
        plataforma.go           ← interface Plataforma (implementada em Kotlin)
        bridge_test.go
      web/
        ponte-wails.js          ← cria window.go.* sobre a interface Kotlin
      app/                      ← módulo Gradle (Kotlin)
        build.gradle.kts
        src/main/
          AndroidManifest.xml
          java/.../MainActivity.kt
          java/.../ponte/PonteJs.kt
          java/.../impressao/{Impressora.kt, ImpressoraSdk*.kt, ImpressoraEscPos.kt, ImpressoraImagem.kt}
          java/.../leitor/{LeitorBroadcast.kt, LeitorCamara.kt}
          assets/web/           ← GERADO (admin/ e pdv/), no .gitignore
          libs/gobridge.aar     ← GERADO, no .gitignore
      build.gradle.kts  settings.gradle.kts  gradle/  gradlew
      scripts/
        build-go.sh             ← gomobile bind → app/libs/gobridge.aar
        build-web.sh            ← wails generate + vite build (modo android) → assets/web
  Makefile                      ← + alvos android-aar, android-web, android-apk, android-install
```

Acrescentar ao `.gitignore`: `cmd/android/app/libs/*.aar`, `cmd/android/app/src/main/assets/web/`, `cmd/android/**/build/`, `cmd/android/.gradle/`, `local.properties`, `*.jks`.

---

## 3. Alterações no código partilhado (`backend/`)

São poucas e todas se testam com `go test`. O desktop continua a funcionar como hoje.

| # | Onde | Alteração | Porquê |
|---|---|---|---|
| B1 | `backend/config/dbpath.go` | Juntar os 4 `os.UserConfigDir()` numa função `diretorioBase()` e acrescentar `config.DefinirDiretorioBase(caminho)` | Em Android, `os.UserConfigDir()` falha (não há `$HOME`). O Kotlin passa `context.filesDir` |
| B2 | `backend/export/abrir*.go` | Acrescentar o gancho `export.DefinirAbridor(func(caminho string) error)`. Sem gancho, mantém-se o comportamento atual | Em Android, "abrir PDF/Excel" passa a ser um *Intent* de visualizar/partilhar |
| B3 | `backend/admin/admin_app.go` → `EscolherFicheiroExcel` | Passar a usar o gancho `EscolhedorFicheiros`. Por omissão continua a usar `runtime.OpenFileDialog` | O diálogo do Wails não existe em Android. O seletor de ficheiros do Android copia o ficheiro para cache e devolve o caminho |
| B4 | `backend/admin/pdv_service.go` | Em Android, `AbrirPDV` deixa de lançar processo. A navegação para o PDV é feita no frontend (§5.2) | Não há segundo executável |
| B5 | `backend/export/talao_texto.go` (**novo**) | `GerarTalaoTexto(d DadosRecibo, colunas int) []LinhaTalao`: linhas já formatadas para 32 colunas (58 mm) ou 48 colunas (80 mm), com marcas de alinhamento, negrito e corte | É a única fonte do layout do talão impresso. Kotlin só envia para a impressora. Fica testável em Go e serve depois para ESC/POS no desktop (SDD §10) |
| B6 | `backend/api/app.go` + `admin_app.go` | Novo método `ObterTalaoImpressao(vendaID, colunas)` | O frontend pede o talão e chama a impressora nativa |

---

## 4. A ponte Go ↔ JavaScript

**Go (`gobridge`)**, a API mínima que o gomobile aceita (só strings e erros):

```go
type Plataforma interface {          // implementada em Kotlin
    AbrirFicheiro(caminho string) error
    EscolherFicheiro(filtro string) (string, error)
}

func Iniciar(dirDados string, p Plataforma) error   // Fx: admin.Module + CoreModule; chama OnStartup
func Chamar(alvo, metodo, argsJSON string) (string, error)
// alvo ∈ {"admin.AdminApp", "api.App"}; argsJSON = "[arg1, arg2, …]"
// devolve JSON do 1.º valor de retorno; erro Go → erro (texto) igual ao do Wails
```

- **Despacho por reflexão:** procura o método pelo nome, faz `json.Unmarshal` de cada argumento para o tipo do parâmetro, chama-o e converte o resultado. Os métodos não exportados e o `OnStartup` ficam bloqueados.
- **`api.App` do PDV:** abre-se através do `admin.AbrirLoja(caminhoDaLojaÚnica)` existente, depois do primeiro arranque, e fica aberta enquanto a app corre. O desktop já tem as duas ligações ao mesmo `loja.db` (processos admin e app), por isso não é uma situação nova.
- **Kotlin (`PonteJs`):** `@JavascriptInterface fun chamar(id, alvo, metodo, args)` corre o `Chamar` numa coroutine `Dispatchers.IO` e responde com `evaluateJavascript("__ponte.resolver(id, ok, json)")`. A UI nunca bloqueia.
- **JS (`ponte-wails.js`):** cria `window.go = { admin: { AdminApp: Proxy }, api: { App: Proxy } }`. Cada chamada devolve uma `Promise` resolvida ou rejeitada pelo `id`. Também define `window.__PLATAFORMA = "android"`.
- **Injeção do script:** o `WebViewAssetLoader` serve `https://appassets.androidplatform.net/…`. Um handler acrescenta `<script src="/ponte-wails.js">` ao `index.html` antes do bundle. Não é preciso alterar o `index.html` dos frontends.

---

## 5. Frontends em modo Android

### 5.1 Build

- `scripts/build-web.sh` corre `wails generate module` nos dois `cmd/*` (o mesmo passo que foi preciso hoje para compilar os frontends no WSL) e depois `vite build --mode android --base ./`. O resultado vai para `assets/web/admin` e `assets/web/pdv`.
- **Alvo de browser:** nos terminais POS sem Google Play, a WebView do sistema pode ser antiga. O Vite 7 compila por omissão para Chrome ≥ 107. Em modo android, usar `build.target: "chrome80"`, ou `@vitejs/plugin-legacy` se o spike mostrar uma WebView mais antiga.

### 5.2 Comportamento específico (atrás de `window.__PLATAFORMA === "android"`)

- **Admin → PDV:** o botão "PDV" navega para `/pdv/index.html` em vez de chamar `AbrirPDV`. O PDV ganha o botão "Voltar à administração". O botão "Voltar" do Android usa o histórico da WebView.
- **Imprimir talão:** o `ReciboModal` do PDV e o detalhe de venda do admin usam `ObterTalaoImpressao` + `window.Impressora.imprimir(...)` em vez de `window.print()`. Fica ainda a opção "Partilhar PDF" para enviar por WhatsApp ou email.
- **Leitor de códigos:** novo hook `useLeitorCodigo(callback)` partilhado pelos dois frontends. Ouve o evento `codigo-lido`, que o Kotlin emite quando o leitor está em modo broadcast. Se o leitor estiver em modo teclado, o comportamento atual já funciona (a Receção já trata "Enter com código exato").
  - PDV: acrescenta o produto ao carrinho e soma +1 se já lá estiver.
  - Inventário: posiciona no item e soma +1 à contagem.
  - Stock/Produtos: abre o produto.
  - Receção: acrescenta a linha (como hoje).

### 5.3 Layout para ecrã pequeno (é a maior fatia do trabalho)

O admin (~8 500 linhas TSX) foi desenhado para 1280×800. Num ecrã de 6" em retrato:

- Breakpoint `≤ 600px`: a barra lateral passa a menu gaveta + barra inferior com 4–5 atalhos (Vender, Stock, Receção, Inventário, Mais).
- As tabelas largas passam a lista de cartões (componente `TabelaResponsiva` reutilizável). Os modais passam a ecrã inteiro.
- Alvos de toque ≥ 44 px, e campos numéricos com `inputmode="decimal"`.
- **Prioridade dos ecrãs:**
  1. PDV venda + talão
  2. Login/Wizard
  3. Stock (consulta)
  4. Receção
  5. Inventário
  6. Vendas/recibos
  7. Dashboard
  8. Clientes/dívidas
  9. Restantes: Relatórios, Fichas, Cotações, Faturas, Importar Excel. Têm de funcionar, mas o polimento fica para o fim.
- O desktop não pode ficar visualmente pior: todas as regras novas ficam dentro de media queries.

---

## 6. Impressora e leitor

### 6.1 Impressora (`Impressora.kt`)

Interface `Impressora { estado(); imprimir(linhas: List<LinhaTalao>); avançarPapel() }` com três implementações, escolhidas automaticamente e com escolha manual nas Configurações:

1. **Serviço de impressão Sunmi** (`com.sunmi:printerlibrary`, `InnerPrinterManager` → `SunmiPrinterService`). É a implementação principal no V2 Pro: `printText` aceita Unicode (acentos sem code page), tem `printBitmap`, `lineWrap`, negrito e alinhamento, e `updatePrinterState()` indica sem papel, tampa aberta ou sobreaquecimento. Usar com `enterPrinterBuffer`/`commitPrinterBuffer` para o talão sair num só bloco.
2. **ESC/POS cru**: a impressora interna exposta como Bluetooth "InnerPrinter" ou por porta série. Converte acentos para a code page PC860/PC850.
3. **Imagem**: desenha as linhas num `Bitmap` de 384 px (58 mm) e imprime como raster. Resolve qualquer problema de acentos (ç, ã, é) e é o modo de recurso.

O layout é sempre o do Go (B5). Kotlin não decide colunas nem formatação.

### 6.2 Leitor (`leitor/`)

1. **Modo teclado** (wedge): não precisa de código, só de configurar o terminal para enviar um sufixo Enter.
2. **Modo broadcast**: `BroadcastReceiver` com a action e o extra configuráveis. Na Sunmi o valor por omissão é `com.sunmi.scanner.ACTION_DATA_CODE_RECEIVED` com o extra `data` (confirmar no A0). Emite o evento `codigo-lido` para o JS. É o preferido porque funciona sem campo focado. Só se aplica se a unidade tiver cabeça de leitura.
3. **Câmara**: no V2 Pro, usar primeiro o **leitor de câmara da Sunmi**, que já vem no sistema e se abre por *Intent* com o resultado devolvido à Activity. Se não estiver disponível, usar ML Kit Barcode na versão *bundled* (não precisa de Google Play Services). Abre-se num botão 📷 junto aos campos de pesquisa e, no PDV, num botão grande "Ler código".

---

## 7. Fases

Estimativas para 1 programador a tempo inteiro.

### Fase A0 — Spike de viabilidade (3–5 dias) ⚠️ fazer primeiro
**Objetivo:** provar no terminal real os pontos que podem obrigar a mudar o plano.
- `gomobile bind -target=android/arm64,android/arm` de um `gobridge` mínimo que abre o `loja.db`, corre as migrations e devolve `ListarCategorias()` em JSON.
- Confirmar que o `modernc.org/sqlite` compila e corre em `android/arm64` e `android/arm`. **Plano B:** `mattn/go-sqlite3` com CGO via NDK. Só muda o driver em `repository/sqlite.go` e `admin_sqlite.go`.
- Recolher do V2 Pro por `adb`:
  ```
  adb shell getprop ro.build.version.release        # versão do Android
  adb shell getprop ro.product.cpu.abilist          # ABIs (32 ou 64 bits)
  adb shell dumpsys package com.android.webview | findstr versionName   # versão da WebView
  adb shell pm list packages | findstr -i "sunmi scan"   # serviços de impressora/leitor
  ```
- Carregar o build atual do admin com a ponte numa Activity mínima. Se a WebView for anterior ao Chrome 80, ativar já o `@vitejs/plugin-legacy`.
- Imprimir "Olá ç ã é — Talão de teste" pelo `SunmiPrinterService` e ler um código de barras pela câmara (leitor Sunmi).

**Aceitação:** login no admin num terminal real, e o talão de teste sai com acentos corretos.

### Fase A1 — Projeto Android e ponte (5–7 dias)
- Criar `cmd/android/` (Gradle, Kotlin, minSdk 24, targetSdk atual, ABIs `arm64-v8a` + `armeabi-v7a`).
- `gobridge` completo (despacho por reflexão + erros) e testes Go que chamam todos os métodos públicos de `AdminApp` e `api.App` pela ponte e comparam com a chamada direta.
- `ponte-wails.js`, `PonteJs.kt`, `WebViewAssetLoader` com injeção do script.
- Scripts `build-go.sh` / `build-web.sh` + alvos no `Makefile`.

**Aceitação:** o admin completo corre no terminal sem alterações ao React, tal como está hoje no desktop (ainda com o layout de desktop).

### Fase A2 — Integração com a plataforma (3 dias)
- B1 (diretório base), B2 (abrir ficheiro → `FileProvider` + `ACTION_VIEW`/partilhar), B3 (seletor de ficheiros do Android para o Excel), B4 (PDV por navegação).
- Navegação admin ↔ PDV e botão "Voltar".

**Aceitação:** primeiro arranque (wizard) → criar produto → vender no PDV → abrir o recibo PDF → importar um Excel escolhido no telemóvel.

### Fase A3 — Impressão de talões (5–8 dias)
- B5 + B6 em Go, com testes (58 mm / 80 mm, nomes longos, descontos, vários pagamentos, acentos).
- `Impressora` com as 3 implementações e o ecrã "Impressora" nas Configurações (escolher modo, largura, imprimir teste).
- Tratamento de erros: sem papel, tampa aberta, sobreaquecimento. Mostrar a mensagem e não perder a venda, porque a venda já está gravada antes de imprimir.

**Aceitação:** talão impresso automaticamente no fim da venda (configurável) e reimpressão a partir do detalhe da venda.

### Fase A4 — Leitor de códigos de barras (3–4 dias)
- `LeitorBroadcast` configurável + `LeitorCamara` (ML Kit bundled).
- Hook `useLeitorCodigo` e integração no PDV, Inventário, Stock e Receção.
- Som/vibração de confirmação. Para código desconhecido, mostrar um aviso claro com a opção "criar produto com este código".

**Aceitação:** venda de 10 artigos só com o leitor, sem tocar no ecrã, e inventário de um armazém feito por leitura.

### Fase A5 — Layout para ecrã pequeno (10–15 dias)
- Pela ordem de prioridade do §5.3. Componentes partilhados primeiro (`TabelaResponsiva`, `Modal` em ecrã inteiro, navegação).
- Testar também em retrato e paisagem, e no desktop para confirmar que nada regride.

**Aceitação:** os 8 ecrãs prioritários usam-se com uma mão num ecrã de 6", e o desktop fica visualmente igual.

### Fase A6 — Cópias de segurança (3 dias) ⚠️ essencial com dados só no terminal
- "Exportar cópia": `VACUUM INTO` para um ficheiro com data, depois partilhar ou guardar (pen USB-C, Drive, WhatsApp).
- "Restaurar cópia": validar o ficheiro, fechar as ligações, substituir e reabrir. Pede confirmação dupla.
- Lembrete diário no fecho de caixa ("última cópia há X dias").
- Também permite levar a loja do terminal para o PC (abrir o `.db` no admin desktop).

**Aceitação:** exportar → apagar dados da app → restaurar → tudo volta igual.

### Fase A7 — Empacotamento, CI e distribuição (3–4 dias)
- Assinatura release (keystore fora do git, em secret do CI), `versionCode` derivado da versão do `wails.json`.
- Novo job `package-android` no `.github/workflows/ci.yml`: setup-java + Android SDK/NDK + `gomobile init` + `make android-apk`. Gera o APK como artefacto.
- Distribuição por APK (instalação manual/USB). Os terminais POS normalmente não têm Play Store. Opcional: modo quiosque (`startLockTask`) para o operador não sair da app.

**Aceitação:** APK assinado gerado pelo CI e instalado num terminal limpo.

### Fase A8 — Testes no terreno (5 dias)
- Uma loja-piloto durante uma semana, com checklist no §8.
- Medir arranque a frio, tempo por venda, talões por rolo e bateria num turno completo.

**Total estimado: ~8–10 semanas.** A0 decide se o resto se mantém.

---

## 8. Testes

| Camada | Como |
|---|---|
| Go (backend + ponte) | `go test ./...` como hoje. Novos testes para B1–B6 e para a ponte (todos os métodos, JSON de ida e volta, erros) |
| Frontend | `npm run lint` + `npm run build` nos dois modos (desktop e android) no CI |
| Android | Testes instrumentados mínimos: arranca, a ponte responde, formatação do talão em imagem |
| Manual no terminal | Checklist: primeiro arranque · venda à vista/crédito · anulação · talão 58 mm · reimpressão · sem papel a meio · leitor (broadcast e câmara) · inventário · receção · relatório + partilhar PDF · importar Excel · backup/restauro · app morta a meio de uma venda (sem dados corrompidos) · rodar ecrã |

---

## 9. Riscos

| Risco | Impacto | Mitigação |
|---|---|---|
| `modernc.org/sqlite` não corre em Android ARM | Alto | Spike A0. Plano B `mattn/go-sqlite3` (só muda o driver) |
| WebView 62 no V2 Pro (confirmado) | **Alto** | Converter o CSS/JS no build só para o modo android (§0.2, opção 1) e validar no terminal. Atualizar a WebView por adb fica como opcional |
| SDK de impressora proprietário ou mal documentado | Médio | 3 implementações (§6.1). O modo imagem funciona quase sempre |
| Perda de dados (terminal roubado, avariado ou reposto) | **Alto** | Fase A6 com lembretes. A sincronização com o PC fica como evolução futura |
| Dois terminais na mesma loja = duas bases separadas | Médio | Deixar claro: **1 terminal = 1 loja** nesta fase. Para vários terminais partilharem dados é preciso um servidor ou sincronização (SDD §8.5, "nota de evolução futura") |
| Tamanho do APK (~25–40 MB com 2 ABIs) | Baixo | Aceitável. Se for preciso, gerar um APK por ABI |
| UI do admin pouco usável em 6" | Médio | Fase A5 com prioridades. Ecrãs secundários aceitam "funciona, mas é melhor no PC" |

---

## 10. Perguntas em aberto

1. ~~Modelo do terminal~~ → **Sunmi V2 Pro** (§0.1). A Sunmi publica o SDK da impressora no Maven, por isso não é preciso pedir nada ao fornecedor.
2. ~~Largura do papel~~ → 58 mm (impressora interna do V2 Pro). Os outros terminais da loja também vão ser V2 Pro, ou haverá modelos com 80 mm?
3. Versão exata do Android e da WebView, e se há cabeça de leitura: resolve-se com os comandos `adb` da Fase A0.
4. O terminal **substitui o PC** nessa loja, ou também vai haver lojas com PC e terminal ao mesmo tempo? Se for o segundo caso, a sincronização sobe de prioridade.
5. O talão deve levar **logótipo** impresso (imagem da identidade da loja)?
6. Há interesse em **pagamentos por cartão/NFC** do próprio terminal? Fica fora deste plano, mas a arquitetura permite acrescentar depois (é mais uma implementação nativa chamada pelo JS).
