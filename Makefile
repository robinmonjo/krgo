HARDWARE=$(shell uname -m)

build:
	go build

release:
	mkdir -p release
	GOOS=linux go build -o ./release/cargo
	cd release && tar -zcf cargo_$(HARDWARE).tgz cargo

	rm release/cargo

test:
	go install
	go test

clean:
	rm -rf release
