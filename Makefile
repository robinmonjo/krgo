GOPATH:=`pwd`/vendor:$(GOPATH)
VERSION:=1.4.1
HARDWARE=$(shell uname -m)

build: vendor
	GOPATH=$(GOPATH) go build

test: vendor
	GOPATH=$(GOPATH) go install
	GOPATH=$(GOPATH) go test

release:
	mkdir -p release
	GOPATH=$(GOPATH) GOOS=linux go build -o release/cargo
	cd release && tar -zcf cargo-v$(VERSION)_$(HARDWARE).tgz cargo
	rm release/cargo

clean:
	rm -rf ./cargo ./rootfs ./release

vendor:
	sh vendor.sh
