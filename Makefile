VERSION :=$(shell git describe --tags --always)
LDFLAGS := "-s -w -X main.version=$(VERSION)"
OUT_DIR := dist
CMD := ./cmd/rmfakecloud
BINARY := rmfakecloud
# Default build is pure-Go (CGO disabled) so it cross-compiles cleanly. These
# binaries use the pdf_stub.go path and return a clear error for v6/Cairo PDF
# export. Cairo PDF rendering requires CGO + libcairo and only builds natively.
BUILD = CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -o $(@) $(CMD)
BUILD_CAIRO = CGO_ENABLED=1 go build -tags cairo -ldflags $(LDFLAGS) -o $(@) $(CMD)
ASSETS = ui/dist
GOFILES := $(shell find . -iname '*.go' ! -iname "*_test.go")
GOFILES += $(ASSETS)
UIFILES := $(shell find ui/src)
UIFILES += $(shell find ui/public)
UIFILES += ui/package.json
TARGETS := $(addprefix $(OUT_DIR)/$(BINARY)-, x64 armv6 armv7 arm64 win64 docker)
PNPM	= cd ui; pnpm

.PHONY: all run runui clean test testgo testui build-cairo

build: $(OUT_DIR)/$(BINARY)-x64

build-cairo: $(OUT_DIR)/$(BINARY)-cairo-x64

all: $(TARGETS)

$(OUT_DIR)/$(BINARY)-x64:$(GOFILES)
	GOOS=linux $(BUILD_CAIRO)

$(OUT_DIR)/$(BINARY)-armv6:$(GOFILES)
	GOARCH=arm GOARM=6 $(BUILD)

$(OUT_DIR)/$(BINARY)-armv7:$(GOFILES)
	GOARCH=arm GOARM=7 $(BUILD)

$(OUT_DIR)/$(BINARY)-win64:$(GOFILES)
	GOOS=windows $(BUILD)

$(OUT_DIR)/$(BINARY)-arm64:$(GOFILES)
	GOARCH=arm64 $(BUILD)

# Built natively (linux/amd64 on CI) with Cairo, then packaged into the
# debian-based runtime image which ships libcairo2.
$(OUT_DIR)/$(BINARY)-docker:$(GOFILES)
	GOOS=linux $(BUILD_CAIRO)

# Cairo-enabled builds (native rmc-go support).
# NOTE: cairo-arm64 cross-compiles CGO and requires an aarch64 cross toolchain
# plus arm64 libcairo (e.g. via a per-arch container build); it will fail on a
# plain amd64 host without one.
$(OUT_DIR)/$(BINARY)-cairo-x64:$(GOFILES)
	GOOS=linux $(BUILD_CAIRO)

$(OUT_DIR)/$(BINARY)-cairo-arm64:$(GOFILES)
	GOOS=linux GOARCH=arm64 $(BUILD_CAIRO)

container: $(OUT_DIR)/$(BINARY)-docker
	docker build -t rmfakecloud -f Dockerfile.make .
	
run: $(ASSETS)
	go run -tags cairo $(CMD) $(ARG)

$(ASSETS): $(UIFILES) ui/pnpm-lock.yaml
	#@cp ui/node_modules/pdfjs-dist/build/pdf.worker.js ui/public/
	$(PNPM) build
	@#remove unneeded stuff, todo: eject
	@rm ui/build/service-worker.js ui/build/precache-manifest* ui/build/asset-manifest.json 2> /dev/null || true

ui/pnpm-lock.yaml: ui/node_modules ui/package.json
	$(PNPM) i
	@touch -mr $(shell ls -Atd $? | head -1) $@

ui/node_modules:
	mkdir -p $@

runui: ui/pnpm-lock.yaml
	$(PNPM) run dev

clean:
	rm -f $(OUT_DIR)/*
	rm -fr $(ASSETS)

test: testui testgo

testui:
	echo "TODO: fix this"
	#CI=true $(PNPM) test

testgo:
	go test -tags cairo ./...

