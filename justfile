#!/usr/bin/env just --justfile

go-mod := `go list`

test:
    go test ./...

generate:
    go generate ./...

update:
    go get -u
    go mod tidy -v

install-cmd:
    @echo "Installing lua-provider-gen"
    go install ./cmd/lua-provider-gen

    # unavailable on original repo
    # @echo "Installing lua-provider-probe"
    # go install ./cmd/lua-provider-probe

publish tag:
    GOPROXY=proxy.golang.org go list -m {{go-mod}}@{{tag}}
