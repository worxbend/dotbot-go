default:
    @just --list

fmt:
    gofmt -w .

tidy:
    go mod tidy

test:
    go test ./...

vet:
    go vet ./...

build:
    mkdir -p bin
    go build -buildvcs=false -o bin/dotbot ./cmd/dotbot-go

build-linux:
    mkdir -p dist
    GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags="-s -w" -o dist/dotbot-linux-amd64 ./cmd/dotbot-go
    GOOS=linux GOARCH=arm64 go build -buildvcs=false -trimpath -ldflags="-s -w" -o dist/dotbot-linux-arm64 ./cmd/dotbot-go

package-linux: build-linux
    cp dist/dotbot-linux-amd64 dist/dotbot
    tar -C dist -czf dist/dotbot-linux-amd64.tar.gz dotbot
    cp dist/dotbot-linux-arm64 dist/dotbot
    tar -C dist -czf dist/dotbot-linux-arm64.tar.gz dotbot
    rm dist/dotbot
    cd dist && sha256sum dotbot-linux-amd64.tar.gz dotbot-linux-arm64.tar.gz > checksums.txt

verify: fmt tidy test vet build
