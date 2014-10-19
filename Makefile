HARDWARE=$(shell uname -m)

build:
	go build

release:
	mkdir -p release
	GOOS=linux go build -o release/dlrootfs
	cd release && tar -zcf dlrootfs_$(HARDWARE).tgz dlrootfs

	rm release/dlrootfs

test:
	go install
	go test

clean:
	rm -rf release
