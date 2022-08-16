BUILDSTAMP := $(shell date -I)-$(shell git rev-parse --short HEAD)

all:
	CGO_ENABLED=0 go build -o bin/gosma -ldflags "-X main.buildstamp=$(BUILDSTAMP)" .
