SHELL := /bin/bash

TAG=`git describe --exact-match --tags $(git log -n1 --pretty='%h') 2>/dev/null || git rev-parse --abbrev-ref HEAD`

all: build test
tag:
	echo $(TAG)
clean:
	rm -fv bin/contentserve*
build: clean
	go build -o bin/contentserver
build-arch: clean
	GOOS=linux GOARCH=amd64 go build -o bin/contentserver-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -o bin/contentserver-darwin-amd64
build-docker: clean build-arch
	curl https://curl.haxx.se/ca/cacert.pem > .cacert.pem
	docker build -q . > .image_id
	docker tag `cat .image_id` docker-registry.bestbytes.net/contentserver:$(TAG)
	echo "# tagged container `cat .image_id` as docker-registry.bestbytes.net/contentserver:$(TAG)"
	rm -vf .image_id .cacert.pem

package: build
	pkg/build.sh
test:
	go test ./...
