.PHONY: build

default: build

build:
	go build -ldflags '-s -w' -o exe/protobuf-thrift exe/main.go
