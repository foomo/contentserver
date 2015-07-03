SHELL := /bin/bash

options:
	echo "you can clean | test | build | run"
clean:
	rm -f bin/content-server
build:
	make clean
	go build -o bin/content-server
package: clean build
	cli/package.sh
test:
	go test -v  github.com/foomo/contentserver/server/repo
