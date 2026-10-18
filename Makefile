CGO_ENV := CGO_ENABLED=1
TAGS :=
GO := $(CGO_ENV) go

ADDR ?= 127.0.0.1:8080
DB   ?= database/sudoku.db

FUZZ_PARALLEL ?= 4
FUZZ_TIME ?= 20s

MUTATE_PKGS ?= internal/config,internal/logging,internal/sudoku,internal/matmul,internal/solver,internal/policy,internal/puzzle,internal/app,internal/store,internal/server,internal/cli,internal/coverbadge
MUTATE_PARALLEL ?= 8

E2E_TIMEOUT ?= 15m

MEMSOFT ?= 4GiB

GOMEMLIMIT ?= 512MiB

ifeq ($(OS),Windows_NT)
EXE := .exe
endif

RACE_PKGS ?= ./internal/solver ./internal/store ./internal/server ./internal/cli ./internal/e2e
RACE_EXIST := $(foreach p,$(RACE_PKGS),$(if $(wildcard $(p)/*.go),$(p),))

NPM_REGISTRY ?= https://registry.npmjs.org
UNPKG ?= https://unpkg.com
HTMX_VERSION ?= 4.0.0
HTMX_SHA256 ?= e484d9171a9db30a39c8f16e3d709d4137f3211c659f8e6125816635033d593f
HTMX_URL ?= $(UNPKG)/htmx.org@$(HTMX_VERSION)/dist/htmx.min.js

PLANTUML_VERSION ?= 1.2026.8
PLANTUML_SHA256 ?= 5e1ecfa8ecd32c90b03bbf3b1eb6f020943f98ab0fcf4032be31a0002ee2c462
PLANTUML_URL ?= https://github.com/plantuml/plantuml/releases/download/v$(PLANTUML_VERSION)/plantuml-$(PLANTUML_VERSION).jar
PLANTUML_JAR ?= .tools/plantuml-$(PLANTUML_VERSION).jar

ifeq ($(OS),Windows_NT)
GCC_UCRT_BIN ?= $(subst \,/,$(USERPROFILE))/scoop/apps/mingw-winlibs-ucrt/current/bin
ifneq ($(origin CC),command line)
CC := $(GCC_UCRT_BIN)/gcc.exe
endif
ifneq ($(origin CXX),command line)
CXX := $(GCC_UCRT_BIN)/g++.exe
endif
export CC CXX
endif

BIN := dist/sudoku$(EXE)

FZ := $(CGO_ENV) GOMEMLIMIT=$(MEMSOFT) go test $(TAGS)

RELEASE_FLAGS := -trimpath -ldflags "-s -w"

.PHONY: help all build test test-pkg test-one race vet fmt lint lint-pkg typecheck sloc fuzz fuzz-one mutate memtest e2e verify vendor-assets diagrams gen train run server seed tidy clean coverage screenshots release topics release-upload

help: ## show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target>\n\nTargets:\n"} \
	      /^[a-zA-Z_-]+:.*##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

all: test ## run the full test suite (default)

build: ## build the sudoku binary into dist/
	$(GO) build $(TAGS) -o $(BIN) ./cmd/sudoku

test: ## run the full test suite
	$(CGO_ENV) GOMEMLIMIT=$(MEMSOFT) go test $(TAGS) ./...

test-pkg: ## run one package's tests: make test-pkg PKG=./internal/config
	$(CGO_ENV) GOMEMLIMIT=$(MEMSOFT) go test $(TAGS) $(PKG)

test-one: ## run one named test: make test-one PKG=./internal/cli TEST=TestMainBare
	$(CGO_ENV) GOMEMLIMIT=$(MEMSOFT) go test $(TAGS) -run '^$(TEST)$$' -count=1 -v $(PKG)

race: ## go test -race over the concurrent core (RACE_PKGS)
	$(CGO_ENV) GOMEMLIMIT=$(MEMSOFT) go test $(TAGS) -race -count=1 $(RACE_EXIST)

vet: ## go vet
	$(GO) vet $(TAGS) ./...

fmt: ## gofumpt all Go files
	gofumpt -w .

lint: ## golangci-lint run
	golangci-lint run

lint-pkg: ## lint one package: make lint-pkg PKG=./internal/config
	golangci-lint run $(PKG)

typecheck: ## compile every package, then go vet
	$(GO) build $(TAGS) ./...
	$(GO) vet $(TAGS) ./...

sloc: ## enforce the 1000-line ceiling on every Go file
	$(CGO_ENV) go run ./tools/sloc

fuzz: ## fuzz every target in the repo for FUZZ_TIME each
	$(FZ) -run '^$$' -fuzz FuzzParseInput -fuzztime $(FUZZ_TIME) -parallel $(FUZZ_PARALLEL) ./internal/sudoku
	$(FZ) -run '^$$' -fuzz FuzzSolverCore -fuzztime $(FUZZ_TIME) -parallel $(FUZZ_PARALLEL) ./internal/solver

fuzz-one: ## fuzz one target: make fuzz-one PKG=./internal/sudoku FUZZ_TARGET=FuzzParseInput
	$(FZ) -run '^$$' -fuzz $(FUZZ_TARGET) -fuzztime $(FUZZ_TIME) -parallel $(FUZZ_PARALLEL) $(PKG)

mutate: ## mutation-test MUTATE_PKGS, every surviving mutant fails the build
	$(CGO_ENV) GOMEMLIMIT=$(MEMSOFT) go run ./tools/mutator -packages $(MUTATE_PKGS) -parallel $(MUTATE_PARALLEL)

memtest: ## solver and e2e suites under a hard GOMEMLIMIT
	$(CGO_ENV) GOMEMLIMIT=$(GOMEMLIMIT) go test $(TAGS) -count=1 -timeout $(E2E_TIMEOUT) ./internal/solver ./internal/e2e

e2e: ## end-to-end suite over the real binary, a live server, and the browser UI
	SUDOKU_E2E_UI=1 $(CGO_ENV) GOMEMLIMIT=$(MEMSOFT) go test $(TAGS) -count=1 -v -timeout $(E2E_TIMEOUT) ./internal/e2e

verify: test memtest fmt lint typecheck sloc fuzz mutate e2e ## run the full quality chain, fail-fast

vendor-assets: ## verify or refresh the vendored htmx pin in web/static
	@set -e; \
	trap 'rm -f web/static/*.tmp' EXIT; \
	if tags=$$(curl -fsSL $(NPM_REGISTRY)/-/package/htmx.org/dist-tags 2>/dev/null); then \
	  newest=$$(printf '%s\n' "$$tags" | sed 's/[^:]*:"\([^"]*\)"/\1\n/g' | grep -E '^[0-9]' | grep -v -- '-' | sort -V | tail -1); \
	else \
	  echo "htmx.org: registry unreachable, verifying the local pin only"; \
	  newest=; \
	fi; \
	if [ -n "$$newest" ] && [ "$$newest" != "$(HTMX_VERSION)" ] && [ "$$(printf '%s\n%s\n' "$(HTMX_VERSION)" "$$newest" | sort -V | tail -1)" = "$$newest" ]; then \
	  curl -fsSL "$(UNPKG)/htmx.org@$$newest/dist/htmx.min.js" -o web/static/htmx.min.js.tmp; \
	  if head -c 1 web/static/htmx.min.js.tmp | grep -q '<'; then \
	    echo "htmx.org: $(UNPKG)/htmx.org@$$newest/dist/htmx.min.js returned HTML, refusing to move the pin"; \
	    exit 1; \
	  fi; \
	  mv web/static/htmx.min.js.tmp web/static/htmx.min.js; \
	  sum=$$(sha256sum web/static/htmx.min.js | cut -d' ' -f1); \
	  sed -i "s/^HTMX_VERSION ?=.*/HTMX_VERSION ?= $$newest/" Makefile; \
	  sed -i "s/^HTMX_SHA256 ?=.*/HTMX_SHA256 ?= $$sum/" Makefile; \
	  grep -q "^HTMX_VERSION ?= $$newest\$$" Makefile || { echo "htmx.org: pin rewrite failed"; exit 1; }; \
	  grep -q "^HTMX_SHA256 ?= $$sum\$$" Makefile || { echo "htmx.org: sha rewrite failed"; exit 1; }; \
	  echo "htmx.org: pin moved $(HTMX_VERSION) -> $$newest (sha256 $$sum)"; \
	else \
	  if ! echo "$(HTMX_SHA256)  web/static/htmx.min.js" | sha256sum -c --status - 2>/dev/null; then \
	    curl -fsSL "$(HTMX_URL)" -o web/static/htmx.min.js.tmp; \
	    if head -c 1 web/static/htmx.min.js.tmp | grep -q '<'; then \
	      echo "htmx.org: $(HTMX_URL) returned HTML, refusing to replace the pinned bytes"; \
	      exit 1; \
	    fi; \
	    mv web/static/htmx.min.js.tmp web/static/htmx.min.js; \
	  fi; \
	  echo "$(HTMX_SHA256)  web/static/htmx.min.js" | sha256sum -c -; \
	fi

diagrams: ## render every docs/diagrams/*.puml to a committed PNG beside it
	@set -e; \
	if [ ! -f "$(PLANTUML_JAR)" ] || ! echo "$(PLANTUML_SHA256)  $(PLANTUML_JAR)" | sha256sum -c --status -; then \
	  mkdir -p $(patsubst %/,%,$(dir $(PLANTUML_JAR))); \
	  curl -fsSL "$(PLANTUML_URL)" -o "$(PLANTUML_JAR)"; \
	fi; \
	echo "$(PLANTUML_SHA256)  $(PLANTUML_JAR)" | sha256sum -c -; \
	if ! java -version >/dev/null 2>&1; then \
	  echo "diagrams: java not found; docker fallback: docker run --rm -v $$(pwd):/w -w /w/docs/diagrams plantuml/plantuml:$(PLANTUML_VERSION) -tpng '*.puml'"; \
	  exit 1; \
	fi; \
	java -jar "$(PLANTUML_JAR)" -failfast2 -overwrite -tpng docs/diagrams/*.puml; \
	for src in docs/diagrams/*.puml; do \
	  png="$${src%.puml}.png"; \
	  if [ ! -f "$$png" ]; then echo "diagrams: render failed for $$src"; exit 1; fi; \
	done

gen: build ## print one generated puzzle in the 9-line input format
	./$(BIN) gen

train: ## regenerate the committed race-order policy table
	$(GO) generate ./internal/policy

run server: ## run the web server (http://$(ADDR))
	$(GO) run $(TAGS) ./cmd/sudoku serve --addr $(ADDR) --db $(DB)

seed: ## delete the database so the next 'make run' re-seeds it
	rm -f $(DB)
	@echo "database removed; the next 'make run' will re-seed it"

tidy: ## go mod tidy
	go mod tidy

clean: ## remove generated artifacts (database, binaries, tools)
	rm -f $(DB) *.exe *.test *.out
	rm -rf dist .tools

coverage: build ## recompute Go test coverage and refresh the readme coverage badge
	$(GO) test $(TAGS) -coverpkg=./... -coverprofile=coverage.out ./...
	./$(BIN) coverage -profile=coverage.out -readme=readme.md
	rm -f coverage.out

screenshots: build ## capture the UI and terminal journeys into project-pictures/
	./$(BIN) screenshot

release: ## cross-compile stripped release binaries with zig cc into dist/release
	@set -e; \
	mkdir -p dist/release; \
	for target in x86_64-linux-gnu x86_64-windows-gnu x86_64-macos aarch64-macos; do \
	  zigarch=$$(printf '%s' "$$target" | cut -d- -f1); \
	  goarch=amd64; [ "$$zigarch" = aarch64 ] && goarch=arm64; \
	  goos=linux; \
	  case $$target in *windows*) goos=windows;; *macos*) goos=darwin;; esac; \
	  name=sudoku-v1.0-$$goos-$$goarch; [ $$goos = windows ] && name=$$name.exe; \
	  echo "release: building $$name"; \
	  CC="zig cc -target $$target" CXX="zig c++ -target $$target" CGO_ENABLED=1 GOOS=$$goos GOARCH=$$goarch \
	    go build $(TAGS) $(RELEASE_FLAGS) -o dist/release/$$name ./cmd/sudoku || exit 1; \
	done; \
	ls -l dist/release

topics: ## set the GitHub repository topic list
	gh api -X PUT repos/lavantien/go-sudoku-solver/topics \
	  -F 'names[]=go' -F 'names[]=sudoku' -F 'names[]=sudoku-solver' \
	  -F 'names[]=constraint-propagation' -F 'names[]=exact-cover' \
	  -F 'names[]=simulated-annealing' -F 'names[]=matrix-multiplication' \
	  -F 'names[]=htmx' -F 'names[]=sqlite' -F 'names[]=single-binary' \
	  -F 'names[]=cli' -F 'names[]=puzzle-solver'

release-upload: ## publish the v1.0 GitHub release with the cross-compiled binaries
	gh release create v1.0 --title v1.0 --notes-file docs/release-notes-v1.0.md dist/release/*
