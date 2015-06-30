SHELL := /bin/bash

options:
	echo "you can clean | test | build | run"
clean:
	rm -f bin/contentserver
build:
	make clean
	go build -o bin/contentserver
test:
	go test -v  github.com/foomo/contentserver/server/repo
