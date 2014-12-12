HARDWARE=$(shell uname -m)

build:
	cd dlrootfs && go build

release:
	mkdir -p release
	cd dlrootfs && GOOS=linux go build -o ../release/dlrootfs
	cd release && tar -zcf dlrootfs_$(HARDWARE).tgz dlrootfs

	rm release/dlrootfs

test:
	cd dlrootfs && go install
	cd integration && go test

clean:
	rm -rf release
