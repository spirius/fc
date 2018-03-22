.PHONY: dist

$(@shell mkdir dist)

DISTROS=darwin_amd64 linux_amd64

BINARIES=$(DISTROS:%=dist/gofc_%)
SOURCES=$(shell find . -not -path './vendor/*' -name '*.go')

all: $(BINARIES)

$(BINARIES): $(SOURCES)
	GOOS=$(word 1,$(subst _, ,$(@:dist/gofc_%=%))) \
	    GOARCH=$(word 2,$(subst _, ,$(@:dist/gofc_%=%))) \
	    go build -v -ldflags '-s -w' -o $@ github.com/spirius/fc/cmd/gofc
