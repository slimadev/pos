# Gestão de Retalho — código

Implementação do sistema descrito em `../SDD_Gestao_Bottlestore_Ferragem.md`, seguindo o plano de fases em `../PLANO_IMPLEMENTACAO.md`. Desenho visual de referência em `../design/`.

## Estrutura

```
backend/    domain, usecase, repository, api, export, config, di — 100% partilhado entre cmd/app e cmd/admin
cmd/app/    executável único de loja (Wails + frontend React embutido)
cmd/admin/  executável de administração universal (Fase 9)
migrations/ migrations SQL (golang-migrate) do schema core + módulos opcionais
```

`backend` é o único sítio onde o `go.mod` do módulo (`gestao-retalho`) importa — `cmd/app` e `cmd/admin` só fazem wiring via `backend/di.CoreModule` e chamam `wails.Run`.

## Pré-requisitos (Linux)

Ubuntu 24.04+ só traz `webkit2gtk-4.1`, não a `4.0` que o Wails procura por omissão — por isso todo build/dev usa a tag `webkit2_41` (já embutida no `Makefile`):

```
sudo apt-get install libgtk-3-dev libwebkit2gtk-4.1-dev
```

## Comandos

```
make dev-app     # wails dev (hot reload)
make build-app   # wails build (produção)
make vet         # go vet
make lint        # golangci-lint (instalar à parte)
```

No frontend (`cmd/app/frontend`): `npm run lint`, `npm run format`, `npm run build`.
