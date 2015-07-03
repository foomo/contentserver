#!/bin/bash

USER="foomo"
NAME="content-server"
URL="http://www.foomo.org"
DESCRIPTION="Serves content tree structures very quickly through a json socket api."
LICENSE="LGPL-3.0"

ARCH="amd64"
VERSION=`bin/content-server --version | sed 's/content-server //'`

PACKAGE=`pwd`/pkg
mkdir -p $PACKAGE

package()
{
	OS=$1
	ARCH=$2
	TYPE=$3
	TARGET=$4

	# create build folder
	BUILD=`pwd`/${NAME}-${VERSION}
	mkdir -p $BUILD/usr/bin
	cp bin/${NAME} $BUILD/usr/bin/.

	# build binary
	GOOS=$OS GOARCH=$ARCH go build -o $BUILD/usr/bin/${NAME} contentserver.go

	# create package
	fpm -s dir \
		-t $TYPE \
		--name $NAME \
		--maintainer $USER \
		--version $VERSION \
		--license $LICENSE \
		--description "${DESCRIPTION}" \
		--architecture $ARCH \
		--package pkg \
		--url "${URL}" \
		-C $BUILD \
		.

	# cleanup
	rm -rf $BUILD

	# push
	package_cloud push $TARGET $PACKAGE/${NAME}_${VERSION}_${ARCH}.${TYPE}
}

package linux amd64 deb foomo/content-server/ubuntu/trusty

#package linux amd64 rpm
