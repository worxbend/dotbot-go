version := `version=$(git describe --tags --always --dirty 2>/dev/null || printf dev); printf %s "${version#v}"`

default:
    @just --list

fmt:
    gofmt -w .

tidy:
    go mod tidy

test:
    go test ./...

test-race:
    go test -race ./...

vet:
    go vet ./...

lint:
    golangci-lint run ./...

lint-fix:
    golangci-lint run --fix ./...

vulncheck:
    GOTOOLCHAIN=go1.26.4+auto go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...

build:
    mkdir -p bin
    go build -buildvcs=false -ldflags="-X dotbot-go/internal/app.Version={{version}}" -o bin/dotbot ./cmd/dotbot-go

build-linux:
    mkdir -p dist
    GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags="-s -w -X dotbot-go/internal/app.Version={{version}}" -o dist/dotbot-linux-amd64 ./cmd/dotbot-go
    GOOS=linux GOARCH=arm64 go build -buildvcs=false -trimpath -ldflags="-s -w -X dotbot-go/internal/app.Version={{version}}" -o dist/dotbot-linux-arm64 ./cmd/dotbot-go

package-linux: build-linux
    cp dist/dotbot-linux-amd64 dist/dotbot
    tar -C dist -czf dist/dotbot-linux-amd64.tar.gz dotbot
    cp dist/dotbot-linux-arm64 dist/dotbot
    tar -C dist -czf dist/dotbot-linux-arm64.tar.gz dotbot
    rm dist/dotbot
    cd dist && sha256sum dotbot-linux-amd64.tar.gz dotbot-linux-arm64.tar.gz > checksums.txt

verify: fmt tidy lint test test-race vet build
