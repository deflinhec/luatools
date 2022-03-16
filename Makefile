# Define
VERSION=1.0.0
BUILD=$(shell git rev-parse HEAD)

# Setup linker flags option for build that interoperate with variable names in src code
LDFLAGS='-s -w -X "main.Version=$(VERSION)" -X "main.Build=$(BUILD)"'

.PHONY: default all build

all: fmt tidy darwin linux windows

default: all

fmt:
	go fmt ./...

tidy:
	go mod tidy

# Sperate "linux-amd64" as GOOS and GOARCH
OSARCH_SPERATOR = $(word $2,$(subst -, ,$1))
# Platform build options
cross-compile-%: export GOOS=$(call OSARCH_SPERATOR,$*,1)
cross-compile-%: export GOARCH=$(call OSARCH_SPERATOR,$*,2)
cross-compile-%: 
	go build -ldflags $(LDFLAGS) -o ./build/$(GOOS)-$(GOARCH)/ ./cmd/...

# Arch build options
arch-%: export GOARCH=$(call OSARCH_SPERATOR,$*,1)
arch-%: fmt tidy
	go build -ldflags $(LDFLAGS) -o ./build/$(GOARCH)/ ./cmd/...

# Local build options
build:
	go build -ldflags $(LDFLAGS) ./cmd/...

linux: cross-compile-linux-amd64 cross-compile-linux-arm64
darwin: cross-compile-darwin-amd64 cross-compile-darwin-arm64
windows: cross-compile-windows-amd64



