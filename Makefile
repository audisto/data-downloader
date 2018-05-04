all: build

ensure-dependency:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/rakyll/statik
	dep ensure

embed-static:
	@echo "embedding web static files into the binary..."
	go generate web/server.go

install: embed-static
	go install ./pkg/* ./cmd/* ./web

build: embed-static
	go build -o bin/audisto-cli-dev ./cmd/audisto-cli/...
	@echo "New binary available at bin/audisto-cli-dev"

test: embed-static
	go test -race ./pkg/downloader ./web ./cmd/audisto-cli

release-windows: embed-static
	GOOS=windows GOARCH=amd64 go build -ldflags '-s -w' -o bin/audisto-cli-windows-amd64.exe ./cmd/audisto-cli/...

release-linux: embed-static
	GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o bin/audisto-cli-linux-amd64 ./cmd/audisto-cli/...

release-macosx: embed-static
	GOOS=darwin GOARCH=amd64 go build -ldflags '-s -w' -o bin/audisto-cli-macosx-amd64 ./cmd/audisto-cli/...

release: embed-static release-macosx release-linux release-windows
