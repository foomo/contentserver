SHELL := /bin/bash

options:
	echo "you can clean | test | build | build-arch | run | package"
clean:
	rm -fv bin/contentserve*
build: clean
	go build -o bin/contentserver
build-arch: clean
	GOOS=linux GOARCH=amd64 go build -o bin/contentserver-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -o bin/contentserver-darwin-amd64
package: build
	pkg/build.sh
test:
	go test -v github.com/foomo/contentserver/server/repo
