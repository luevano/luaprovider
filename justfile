#!/usr/bin/env just --justfile

go-mod := `go list`

test:
    go vet ./...
    go test -race ./...

generate:
    go generate ./...

update:
    go get -u
    go mod tidy -v

install-cmd:
    @echo "Installing lua-provider-gen"
    go install ./cmd/lua-provider-gen

publish tag:
    GOPROXY=proxy.golang.org go list -m {{go-mod}}@{{tag}}
