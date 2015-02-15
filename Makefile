GOPATH:=`pwd`/vendor:$(GOPATH)  #inject vendored package
GOPATH:=`pwd`/vendor/src/github.com/docker/docker/vendor:$(GOPATH) #inject docker vendored package

VERSION:=1.4.1
HARDWARE=$(shell uname -m)

build: vendor
	GOPATH=$(GOPATH) go build

test: vendor build
	GOPATH=$(GOPATH) PATH=$(PATH):`pwd` go test

release:
	mkdir -p release
	GOPATH=$(GOPATH) GOOS=linux go build -o release/krgo
	cd release && tar -zcf krgo-v$(VERSION)_$(HARDWARE).tgz krgo
	rm release/krgo

clean:
	rm -rf ./krgo ./release ./vendor/pkg/*

vendor:
	sh vendor.sh
