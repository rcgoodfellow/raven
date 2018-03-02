all: \
	build/rvn

build/rvn: rvn-cli/rvn.go | build
	go build -o build/rvn rvn-cli/rvn.go

build:
	mkdir build

clean:
	rm -rf build
