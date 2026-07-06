.PHONY: all build dev css css-watch templ templ-watch test deps clean

BINARY := signal
BIN_DIR := bin
TAILWIND_VERSION := v3.4.17
TAILWIND_BIN := $(BIN_DIR)/tailwindcss
TEMPL_BIN := $(BIN_DIR)/templ
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

all: build

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

$(TEMPL_BIN): | $(BIN_DIR)
	go install github.com/a-h/templ/cmd/templ@latest
	cp $(shell go env GOPATH)/bin/templ $(TEMPL_BIN)

$(TAILWIND_BIN): | $(BIN_DIR)
	curl -sL "https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-$(subst darwin,macos,$(GOOS))-$(GOARCH)" -o $(TAILWIND_BIN)
	chmod +x $(TAILWIND_BIN)

deps: $(TEMPL_BIN) $(TAILWIND_BIN)

css: $(TAILWIND_BIN)
	$(TAILWIND_BIN) -i web/static/css/input.css -o web/static/css/main.css --minify

css-watch: $(TAILWIND_BIN)
	$(TAILWIND_BIN) -i web/static/css/input.css -o web/static/css/main.css --watch

templ: $(TEMPL_BIN)
	$(TEMPL_BIN) generate

templ-watch: $(TEMPL_BIN)
	$(TEMPL_BIN) generate --watch

dev:
	$(MAKE) css
	$(MAKE) templ
	go run ./cmd/server -config config.yaml

build: deps
	$(MAKE) css
	$(MAKE) templ
	go build -o $(BINARY) ./cmd/server

test:
	go test ./...

clean:
	rm -f $(BINARY) web/static/css/main.css
	rm -rf $(BIN_DIR)
	find web/templates -name '*_templ.go' -delete
