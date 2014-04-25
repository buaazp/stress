#!/bin/sh

for OS in "linux" "darwin"; do
	for ARCH in "386" "amd64"; do
		GOOS=$OS  CGO_ENABLED=0 GOARCH=$ARCH go build -o stress 
		ARCHIVE=stress-$OS-$ARCH.tar.gz
		tar -czf $ARCHIVE stress
		echo $ARCHIVE
	done
done
