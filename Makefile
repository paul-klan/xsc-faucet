## commom

DATETIME=$(shell date +%Y-%m-%dT%H:%M:%S%z)
PACKAGES=$(shell go list ./... | grep -v /vendor/)
VETPACKAGES=$(shell go list ./... | grep -v /vendor/ | grep -v /examples/)
GOFILES=$(shell find . -name "*.go" -type f -not -path "./vendor/*")
COMMITHASH=$(shell git rev-parse --short HEAD)
BUILDDATE=$(shell TZ=Asia/Shanghai date +%FT%T%z)

MODFILE=go.mod

all: fmt mod build

.PHONY: fmt vet build

list:
	@echo ${DATETIME}
	@echo ${PACKAGES}
	@echo ${VETPACKAGES}
	@echo ${GOFILES}

fmt:
	@gofmt -s -w ${GOFILES}

init:
	@if [ -f ${MODFILE} ] ; then rm ${MODFILE} ; fi
	@go mod init

mod:
	@go mod tidy

vet:
	@go vet $(VETPACKAGES)

clean:
	@rm bin/faucet
	@rm bin/crypt

build:
	@GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/faucet main.go
	@GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/crypt  tools/crypt/main.go

imports:
	@goimports -local xchain -w .
