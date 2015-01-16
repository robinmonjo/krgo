GOPATH:=`pwd`/vendor:$(GOPATH)

build: vendor
	GOPATH=$(GOPATH) go build

test: vendor
	GOPATH=$(GOPATH) go install
	GOPATH=$(GOPATH) go test

clean:
	rm -rf ./cargo ./rootfs

vendor:
	sh vendor.sh
