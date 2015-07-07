SHELL := /bin/bash

options:
	echo "you can clean | test | build | run | package"
clean:
	rm -f bin/content-server
build: clean
	go build -o bin/content-server
package: build
	pkg/build.sh
test:
	go test -v github.com/foomo/contentserver/server/repo
