# Ubuntu 24.04+ (e distros recentes) só trazem webkit2gtk-4.1, não a 4.0 que
# o Wails procura por omissão — daí a tag `webkit2_41` em todo alvo Linux.
# https://wails.io/docs/guides/linux
UNAME := $(shell uname -s)
ifeq ($(UNAME),Linux)
	WAILS_TAGS := -tags webkit2_41
endif

.PHONY: dev-app build-app vet lint

dev-app:
	cd cmd/app && wails dev $(WAILS_TAGS)

build-app:
	cd cmd/app && wails build $(WAILS_TAGS)

vet:
	go vet ./...

lint:
	golangci-lint run ./...
