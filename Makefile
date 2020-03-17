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
	go build -o bin/data-downloader-dev ./cmd/audisto-cli/...
	@echo "New binary available at bin/data-downloader-dev"

test: embed-static
	go test -race ./pkg/downloader ./web ./cmd/audisto-cli

release-windows: embed-static
	GOOS=windows GOARCH=amd64 go build -ldflags '-s -w' -o bin/data-downloader-windows-amd64.exe ./cmd/audisto-cli/...

release-linux: embed-static
	GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o bin/data-downloader-linux-amd64 ./cmd/audisto-cli/...

release-macosx: embed-static
	GOOS=darwin GOARCH=amd64 go build -ldflags '-s -w' -o bin/data-downloader-macosx-amd64 ./cmd/audisto-cli/...

release: embed-static release-macosx release-linux release-windows
