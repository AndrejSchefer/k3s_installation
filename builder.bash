#!/bin/bash

mkdir -p builds

platforms=("linux/amd64" "linux/arm64" "linux/arm" "windows/amd64" "darwin/amd64" "darwin/arm64")

for platform in "${platforms[@]}"
do
  GOOS=${platform%/*}
  GOARCH=${platform#*/}
  output_name="k3s-installer-$GOOS-$GOARCH"
  [ "$GOOS" = "windows" ] && output_name+=".exe"
  
  echo "Building for $GOOS/$GOARCH..."
  CGO_ENABLED=0 go build -ldflags="-s -w" -o builds/$output_name .
done
