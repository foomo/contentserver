SHELL := /bin/bash

TAG?=latest
IMAGE=docker-registry.bestbytes.net/contentserver

# Utils

all: build test
tag:
	echo $(TAG)
dep:
	go mod download && go mod vendor && go install -i ./vendor/...
clean:
	rm -fv bin/contentserve*

# Build

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

build-testclient:
	go build -o bin/testclient -i github.com/foomo/contentserver/testing/client

build-testserver:
	go build -o bin/testserver -i github.com/foomo/contentserver/testing/server

package: build
	pkg/build.sh

# Docker

docker-build:
	docker build -t $(IMAGE):$(TAG) .

docker-push:
	docker push $(IMAGE):$(TAG)

# Testing / benchmarks

test:
	go test ./...

bench:
	go test -run=none -bench=. ./...

run-testserver:
	bin/testserver -json-file var/cse-globus-stage-b-with-main-section.json

run-contentserver:
	contentserver -var-dir var -webserver-address :9191 -address :9999 -log-level debug http://127.0.0.1:1234

clean-var:
	rm var/contentserver-repo-2019*

# Profiling

test-cpu-profile:
	go test -cpuprofile=cprof-client github.com/foomo/contentserver/client
	go tool pprof --text client.test cprof-client

	go test -cpuprofile=cprof-repo github.com/foomo/contentserver/repo
	go tool pprof --text repo.test cprof-repo

test-gctrace:
	GODEBUG=gctrace=1 go test ./...

test-malloctrace:
	GODEBUG=allocfreetrace=1 go test ./...

trace:
	curl http://localhost:6060/debug/pprof/trace?seconds=60 > cs-trace
	go tool trace cs-trace

pprof-heap-web:
	go tool pprof -http=":8081" http://localhost:6060/debug/pprof/heap

pprof-cpu-web:
	go tool pprof -http=":8081" http://localhost:6060/debug/pprof/profile