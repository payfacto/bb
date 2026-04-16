# Version is derived from the nearest git tag, falling back to "dev" when
# there is no tag or git is unavailable.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X 'github.com/payfacto/bb/cmd.Version=$(VERSION)'

.PHONY: build test clean install

build:
	go build -ldflags "$(LDFLAGS)" -o bb .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

clean:
	rm -f bb bb.exe
