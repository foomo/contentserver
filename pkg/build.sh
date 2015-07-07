#!/bin/bash

USER="foomo"
NAME="content-server"
URL="http://www.foomo.org"
DESCRIPTION="Serves content tree structures very quickly through a json socket api."
LICENSE="LGPL-3.0"

# get version
VERSION=`bin/content-server --version | sed 's/content-server //'`

# create temp dir
TEMP=`pwd`/pkg/tmp
mkdir -p $TEMP

package()
{
	OS=$1
	ARCH=$2
	TYPE=$3
	TARGET=$4

	# copy license file
	cp LICENSE $LICENSE

	# define source dir
	SOURCE=`pwd`/pkg/${TYPE}

	# create build folder
	BUILD=${TEMP}/${NAME}-${VERSION}
	#rsync -rv --exclude **/.git* --exclude /*.sh $SOURCE/ $BUILD/

	# build binary
	GOOS=$OS GOARCH=$ARCH go build -o $BUILD/usr/local/bin/${NAME}

	# create package
	fpm -s dir \
		-t $TYPE \
		--name $NAME \
		--maintainer $USER \
		--version $VERSION \
		--license $LICENSE \
		--description "${DESCRIPTION}" \
		--architecture $ARCH \
		--package $TEMP \
		--url "${URL}" \
		-C $BUILD \
		.

	# push
	package_cloud push $TARGET $TEMP/${NAME}_${VERSION}_${ARCH}.${TYPE}

	# cleanup
	rm -rf $TEMP
	rm $LICENSE
}

package linux amd64 deb foomo/content-server/ubuntu/precise
package linux amd64 deb foomo/content-server/ubuntu/trusty

#package linux amd64 rpm
