#!/usr/bin/env bash
set -e

function release() {
  local GOOS=$1
  local GOARCH=$2
  local VERSION=0.1

  echo "$GOOS ($GOARCH)..."
  GOOS=$GOOS GOARCH=$GOARCH go build ./...
  local bin="benchttp"

  if [ ! -e "$bin" ]; then
    bin="benchttp.exe"
  fi

  mkdir -p releases/
  tar czf releases/benchttp-${VERSION}-${GOOS}-${GOARCH}.tgz $bin
  rm $bin
}

release "darwin" "amd64"
release "windows" "386"
release "windows" "amd64"
release "linux" "386"
release "linux" "amd64"
release "linux" "arm"
