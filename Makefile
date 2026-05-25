# lean-mac build / install targets.
#
# Usage:
#   make build      compile ./bin/lm
#   make install    install lm into $(PREFIX)/bin (default: /opt/homebrew/bin)
#   make uninstall  remove the installed binary
#   make tag VERSION=v0.1.0   create + push a git tag for a brew release

PREFIX  ?= /opt/homebrew
BINDIR  ?= $(PREFIX)/bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build install uninstall clean tag test vet

build:
	@mkdir -p bin
	go build -trimpath -ldflags '$(LDFLAGS)' -o bin/lm ./cmd/lm
	@echo "built bin/lm ($(VERSION))"

install: build
	@install -d "$(BINDIR)"
	install -m 0755 bin/lm "$(BINDIR)/lm"
	@echo "installed $(BINDIR)/lm"
	@command -v lm >/dev/null && echo "ready: $$(command -v lm)" || \
		echo "warning: $(BINDIR) is not on your PATH"

uninstall:
	rm -f "$(BINDIR)/lm"
	@echo "removed $(BINDIR)/lm"

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf bin

# Tag + push a release. Homebrew formula's `url` points at the tag tarball.
#   make tag VERSION=v0.1.0
tag:
	@test -n "$(VERSION)" || (echo "VERSION required"; exit 1)
	git tag -a $(VERSION) -m "release $(VERSION)"
	git push origin $(VERSION)
	@echo "tagged $(VERSION). Now run: ./scripts/update-formula-sha.sh $(VERSION)"
