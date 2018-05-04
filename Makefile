all: build

ensure-dependency:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure

embed-static:
	@echo "embedding web static files into the binary..."
	go generate web/server.go

build: embed-static
	go build -o audisto-cli ./cmd/audisto-cli/...

run: build
	./audisto-cli

test: embed-static
	go test -race ./pkg/downloader ./web ./cmd/audisto-cli

release: embed-static release-macosx release-linux release-windows

release-windows: embed-static
	GOOS=windows GOARCH=amd64 go build -ldflags '-s -w' -o bin/audisto-cli-windows-amd64.exe main.go

release-linux: embed-static
	GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o bin/audisto-cli-linux-amd64 main.go

release-macosx: embed-static
	GOOS=darwin GOARCH=amd64 go build -ldflags '-s -w' -o bin/audisto-cli-macosx-amd64 main.go
