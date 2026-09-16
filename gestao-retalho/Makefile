# Ubuntu 24.04+ (e distros recentes) só trazem webkit2gtk-4.1, não a 4.0 que
# o Wails procura por omissão — daí a tag `webkit2_41` em todo alvo Linux.
# https://wails.io/docs/guides/linux
UNAME := $(shell uname -s)
ifeq ($(UNAME),Linux)
	WAILS_TAGS := -tags webkit2_41
endif

# O VSCode no Linux corre como snap e injeta GTK_PATH/GDK_PIXBUF_MODULEDIR/
# GIO_MODULE_DIR a apontar para o GTK3 embutido no PRÓPRIO snap do editor.
# Um binário Wails lançado a partir do terminal integrado do VSCode carrega
# esses módulos em vez dos do sistema, e crasha com "symbol lookup error:
# ...core20.../libpthread.so.0: undefined symbol __libc_pthread_init" — nada
# a ver com a app. Limpar estas variáveis é inofensivo em qualquer outro
# terminal (ficam simplesmente por definir).
UNSET_SNAP_GTK_ENV := env -u GTK_PATH -u GTK_EXE_PREFIX -u GDK_PIXBUF_MODULEDIR \
	-u GDK_PIXBUF_MODULE_FILE -u GIO_MODULE_DIR -u GTK_IM_MODULE_FILE -u LOCPATH

.PHONY: dev-app build-app run-app vet lint

dev-app:
	cd cmd/app && $(UNSET_SNAP_GTK_ENV) wails dev $(WAILS_TAGS)

build-app:
	cd cmd/app && wails build $(WAILS_TAGS)

run-app: build-app
	$(UNSET_SNAP_GTK_ENV) ./cmd/app/build/bin/gestao-retalho

vet:
	go vet ./...

lint:
	golangci-lint run ./...
