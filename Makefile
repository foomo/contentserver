SHELL := /bin/bash

TAG?=latest
IMAGE=docker-registry.bestbytes.net/contentserver

all: build test
tag:
	echo $(TAG)
dep:
	go mod download && go mod vendor && go install -i ./vendor/...
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
	docker tag `cat .image_id` $(IMAGE):$(TAG)
	echo "# tagged container `cat .image_id` as $(IMAGE):$(TAG)"
	rm -vf .image_id .cacert.pem

package: build
	pkg/build.sh

# docker

docker-build:
	docker build -t $(IMAGE):$(TAG) .

docker-push:
	docker push $(IMAGE):$(TAG)

# testing / benchmarks

test:
	go test ./...

bench:
	go test -run=none -bench=. ./...

# profiling

test-cpu-profile:
	go test -cpuprofile=cprof-client github.com/foomo/contentserver/client
	go tool pprof --text client.test cprof-client

	go test -cpuprofile=cprof-repo github.com/foomo/contentserver/repo
	go tool pprof --text repo.test cprof-repo

test-gctrace:
	GODEBUG=gctrace=1 go test ./...

test-malloctrace:
	GODEBUG=allocfreetrace=1 go test ./...