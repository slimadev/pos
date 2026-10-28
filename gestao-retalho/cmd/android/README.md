# App Android (terminal Sunmi V2 Pro)

App Android que hospeda os frontends do admin (`cmd/admin`) e do PDV (`cmd/app`) em WebViews. Plano completo em `../../../PLANO_ANDROID.md`.

O terminal é do PDV; a administração fica atrás de uma password.

- **Arranque.** Com a loja configurada e pelo menos um operador ativo, a `MainActivity` passa logo para a `PdvActivity`: "Quem está a vender?" e o PIN do operador. Pergunta isso a `AdminApp.ObterEstadoArranque`, que não exige sessão.
- **Sem loja ou sem operadores,** abre no admin. Depois do login, sem operadores, vai direto a *Utilizadores* com o aviso "Último passo" e o formulário de novo operador aberto. O próprio PDV recusa vender sem operador identificado: o backend (`exigirTurno`) e o ecrã "Ainda não há operadores" bloqueiam-no.
- **PDV → admin:** o botão **Administração** no cabeçalho do PDV (`Nativo.abrirAdmin()`, visível fora de um turno) abre o admin, sempre no login.
- **Admin → PDV:** os botões **PDV** e **Sair para o PDV** (`Nativo.abrirPDV()`) terminam a sessão do admin e voltam ao PDV. Fechar o admin com "voltar" também termina a sessão.
- O "voltar" no PDV põe a app em segundo plano; nunca abre o admin.

## Como funciona

```
React (admin / PDV)  ──window.go.*──►  web/ponte.js  ──window.Ponte──►  PonteJs.kt
                                                                          │
         ◄──── window.__ponte.resolver(id, ok, json) ────────────────────┘
                                                                          ▼
                                           gobridge.aar (gomobile)  →  backend Go  →  SQLite
```

- **`gobridge/`** (Go, exportado com gomobile): `Iniciar(pasta, fuso, plataforma)` e `Chamar(alvo, método, argsJSON)`. Despacha por reflexão para `admin.AdminApp` ou `api.App`, com a mesma semântica dos bindings do Wails. Os ecrãs não sabem que não estão no desktop.
- **`RetalhoApp.kt`** arranca o backend uma vez, com os dados em `filesDir` e o fuso horário do terminal. Sem o fuso, o Go no Android ficava em UTC.
- **`PlataformaAndroid.kt`** faz o que no desktop é do sistema: abrir PDF/Excel (visualizador ou partilha) e escolher ficheiros (seletor do Android).
- Os dados ficam em `/data/data/com.gestaoretalho.pos/files/` (`gestao-retalho-admin/admin.db` e a base da loja em `gestao-retalho-admin/lojas/`).

## Demonstração

`PONTE=demo bash cmd/android/scripts/build-web.sh` troca o backend por dados de exemplo (sem Go). No ecrã de login aparece um seletor com quatro lojas, cada uma com o seu tema, módulos e produtos:

| Loja | Tipo | O que mostra |
|---|---|---|
| Ferragem Central | ferragem | PDV direto, 18 produtos em 5 categorias |
| Mercearia Dona Ana | mercearia | PDV direto + venda a granel (kg) |
| Farmácia Saúde+ | farmácia | PDV direto + lotes a vencer |
| Bottlestore XPTO | bottlestore | Contas abertas (comandas), pratos com ficha técnica |

As vendas e as contas feitas no PDV ficam guardadas no terminal (localStorage) e aparecem no admin (Vendas, Dashboard). "Repor dados" volta ao estado inicial.

A mesma ponte serve para pré-visualizar o **desktop**: com `window.__DEMO_DESKTOP = true` definido antes de `ponte.js`, os builds normais (`frontend/dist`) correm num browser com estes dados.

## Ícones

Gerados a partir de `design/icones/admin.svg` e `pdv.svg` (na raiz do repositório):

```
cd design/icones && npm install && node gerar.mjs
```

Escreve `cmd/admin/build/appicon.png` e `windows/icon.ico` (exe da administração), o mesmo para `cmd/app` (exe do PDV) e `app/src/main/res/mipmap-*/ic_launcher(_round).png` (Android).

O ícone Android vem de `design/icones/android.svg`. É um disco claro a encher a área, como os ícones do sistema (Play Store, Maps…), porque o launcher do Sunmi os mostra em círculo. O fundo escuro do ícone dos exe sumia no fundo escuro do terminal.

## Outros aparelhos (telemóveis Android modernos)

O mesmo APK traz o backend em 32 bits (`armeabi-v7a`, para o Sunmi) e em 64 bits (`arm64-v8a`): os Xiaomi e outros aparelhos recentes já não correm 32 bits e fechavam a app ao abrir. As bibliotecas vão alinhadas a 16 KB, como o Android 15+ exige. No Android 14+ o leitor regista-se como `RECEIVER_EXPORTED`. No Android 15+ a página respeita as barras do sistema e o teclado (`comMargensDoSistema`).

## Terminal-alvo

Sunmi V2 Pro, Android 7.1.2 (API 25), só 32 bits (`armeabi-v7a`). WebView **Chrome 62**, que não se consegue atualizar. Por isso o frontend tem um build próprio (`vite build --mode android`):

- JS convertido com `@vitejs/plugin-legacy`.
- `gap` convertido em margens/`grid-gap` pelo plugin `web/postcss-gap-chrome62.js`.
- `inset` e restantes funcionalidades convertidos por `postcss-preset-env`.
- Layout de ecrã estreito em `cmd/admin/frontend/src/mobile.css` e `cmd/app/frontend/src/mobile.css`. Só entra abaixo de 600 px, por isso o desktop não muda. No PDV, o carrinho passa a ser um painel aberto pela barra de resumo no fundo.

## Preparar o ambiente (uma vez, sem sudo)

No WSL, `bash cmd/android/scripts/instalar-ambiente.sh` instala em `~/.local/jdk` e `~/Android/Sdk`: JDK 17, Android command-line tools, `platforms;android-34`, `ndk;27.2.12479018`, e `gomobile`/`gobind` (`go install golang.org/x/mobile/cmd/...` + `gomobile init`).

## Compilar e instalar

1. **Backend Go → `app/libs/gobridge.aar`** (WSL):
   ```
   bash cmd/android/scripts/build-go.sh
   ```
2. **Frontends → `app/src/main/assets/web/{admin,pdv}`** (WSL, precisa de node e de `cmd/*/frontend/wailsjs`, gerado com `wails generate module`):
   ```
   bash cmd/android/scripts/build-web.sh
   ```
3. **APK** (Windows, Android SDK + JDK 17+). O Gradle não funciona diretamente em `\\wsl.localhost\...`, por isso o script espelha o projeto para `%LOCALAPPDATA%\gestao-retalho\android-build` e compila lá:
   ```
   powershell -File cmd\android\scripts\build-apk.ps1 -Instalar
   ```
   `-Instalar` instala por `adb` e abre a app no terminal ligado por USB (com *USB debugging* ativo).

## APK de distribuição

`powershell -File cmd\android\scripts\build-apk.ps1 -Release` gera `gestao-retalho\dist\GestaoRetalho-<versão>.apk`, assinado com a chave da empresa. A versão é a do `cmd/admin/wails.json`, a mesma do desktop. O `versionCode` vem dela: 1.2.3 → 10203.

A chave **não está no repositório**: está em `%LOCALAPPDATA%\gestao-retalho\chaves\` (`retalho-distribuicao.jks` + `assinatura.properties`). Também pode vir de `RETALHO_ASSINATURA=<caminho do .properties>`, por exemplo no CI.

> ⚠️ **Guardar uma cópia dessa pasta em local seguro.** O Android só aceita atualizar uma app assinada com a mesma chave. Sem ela, os terminais só podem desinstalar e reinstalar, o que apaga os dados de quem não tiver cópia de segurança.

Um APK de distribuição não instala por cima de um de desenvolvimento, nem o contrário, porque as assinaturas são diferentes.

## Cópias de segurança

Em *Gestão → Cópias de segurança*:
- **Criar cópia** grava um `copia-<data>.retalho`, que é um zip com `admin.db`, as bases das lojas e o `config.json`. No terminal abre logo a partilha.
- **Guardar em…** usa o seletor do Android: cartão SD, pen USB, Transferências ou Drive.
- **Restaurar** troca os dados no arranque seguinte. Os dados antigos ficam em `antes-do-restauro-<data>/`.

Num terminal novo, o ecrã de primeiro arranque tem "Já tem uma cópia de segurança? Restaurar". O dashboard avisa quando não há cópia ou quando a última tem mais de 7 dias.

A extensão `.retalho`, e não `.zip`, existe porque o gestor de ficheiros do Android 7 abria os `.zip` como pastas no seletor. As `.zip` antigas continuam a ser aceites.

## Leitor de códigos (Sunmi)

A cabeça de leitura envia o código por broadcast (`com.sunmi.scanner.ACTION_DATA_CODE_RECEIVED`, extra `data`). O `WebActivity` converte-o no evento JS `codigo-lido`. No modo "teclado" do leitor, `web/ponte.js` deteta as teclas rápidas terminadas em Enter e gera o mesmo evento.

O evento é ouvido por:
- **PDV:** junta o produto ao carrinho.
- **Receção:** acrescenta a linha.
- **Inventário:** soma +1 à contagem (ou as unidades da caixa).
- **Produtos:** pesquisa o código.

Para simular sem leitor:
```
adb shell am broadcast -a com.sunmi.scanner.ACTION_DATA_CODE_RECEIVED --es data 6001234500011
```

## Depurar no terminal

- Erros de JS aparecem numa faixa vermelha no fundo do ecrã (só na ponte de demonstração).
- `adb logcat -s GestaoWeb Retalho GoLog` mostra a consola JS, os erros nativos e o log do Go.
- Apagar os dados do terminal (volta ao primeiro arranque): `adb shell pm clear com.gestaoretalho.pos`.
- No Sunmi, a tecla ESC (`adb shell input keyevent 111`) abre as definições de *Data usage*. Para fechar o teclado use BACK (`keyevent 4`).
