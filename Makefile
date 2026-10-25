.PHONY: build test fmt vet install clean release example

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

build:
	go build -ldflags "-X main.version=$(VERSION)" -o fleet ./cmd/fleet

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	go vet ./...

install:
	go install -ldflags "-X main.version=$(VERSION)" ./cmd/fleet

clean:
	rm -f fleet

example:
	@mkdir -p $(HOME)/.config/agent-fleet
	@if [ ! -f $(HOME)/.config/agent-fleet/fleet.toml ]; then \
		cp fleet.example.toml $(HOME)/.config/agent-fleet/fleet.toml; \
		echo "Wrote $(HOME)/.config/agent-fleet/fleet.toml"; \
	else \
		echo "Already exists: $(HOME)/.config/agent-fleet/fleet.toml"; \
	fi

release:
	mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/fleet-linux-x64 ./cmd/fleet
	GOOS=linux   GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/fleet-linux-arm64 ./cmd/fleet
	GOOS=darwin  GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/fleet-darwin-x64 ./cmd/fleet
	GOOS=darwin  GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/fleet-darwin-arm64 ./cmd/fleet
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/fleet-windows-x64.exe ./cmd/fleet
	GOOS=windows GOARCH=386   go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/fleet-windows-x86.exe ./cmd/fleet
	cd dist && sha256sum * > checksums.txt
