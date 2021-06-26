#!/usr/bin/env bash

set -e 

VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <semantic version>"
    exit 1
fi

# FIXME replace this with goreleaser
OS_ARCH_STATIC="freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm openbsd/386 openbsd/amd64"
OS_ARCH="darwin/amd64 windows/amd64 windows/386"
ASSETS_DIR="./release-bin"
BINARY_NAME="terraform-provider-kubernetes-alpha"

mkdir -p $ASSETS_DIR
rm -rf $ASSETS_DIR/*

gox -ldflags '-d -s -w' -osarch "$OS_ARCH_STATIC" -output "$ASSETS_DIR/{{.Dir}}_${VERSION}_{{.OS}}_{{.Arch}}"
gox -ldflags '-s -w' -osarch "$OS_ARCH" -output "$ASSETS_DIR/{{.Dir}}_${VERSION}_{{.OS}}_{{.Arch}}"

for f in $ASSETS_DIR/*; do
    mv $f $ASSETS_DIR/$BINARY_NAME
    zip -q -j "$f.zip" $ASSETS_DIR/$BINARY_NAME
    rm -f $ASSETS_DIR/$BINARY_NAME
done


echo "Building and signing darwin binary"
GOOS=darwin GOARCH=amd64 go build 
hc-codesign sign -product-name=${BINARY_NAME} ./${BINARY_NAME}
zip -q -j "${ASSETS_DIR}/${BINARY_NAME}_${VERSION}_darwin_amd64.zip" ${BINARY_NAME}

echo "Building and signing windows binaries"
GOOS=windows GOARCH=amd64 go build 
hc-codesign sign -product-name=${BINARY_NAME} ./${BINARY_NAME}.exe
zip -q -j "${ASSETS_DIR}/${BINARY_NAME}_${VERSION}_windows_amd64.zip" ${BINARY_NAME}.exe

GOOS=windows GOARCH=386 go build 
hc-codesign sign -product-name=${BINARY_NAME} ./${BINARY_NAME}.exe
zip -q -j "${ASSETS_DIR}/${BINARY_NAME}_${VERSION}_windows_386.zip" ${BINARY_NAME}.exe
