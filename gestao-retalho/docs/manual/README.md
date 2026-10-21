# Manual do utilizador

`manual.html` é a fonte do manual; as capturas estão em `img/`. O PDF que se
entrega ao cliente gera-se a partir do HTML e não fica no repositório:

    google-chrome --headless=new --no-pdf-header-footer \
      --print-to-pdf=dist/Manual-Gestao-de-Retalho.pdf docs/manual/manual.html

## Refazer as capturas

As capturas usam a loja de demonstração "Ferragem Cossa", criada por
`cmd/_seedmanual` (produtos, stock, clientes com dívidas, vendas dos últimos
14 dias, operadores, cotações e documentos fiscais). Numa pasta de
configuração à parte, para não tocar nos dados reais:

    XDG_CONFIG_HOME=/tmp/manual go run ./cmd/_seedmanual logo.png

Depois abre-se o admin com o mesmo `XDG_CONFIG_HOME` (utilizador `admin`,
palavra-passe `admin123`; PIN da operadora Ana Sitoe: 1234), com a janela
em 1440×900, e captura-se só a janela da aplicação.
