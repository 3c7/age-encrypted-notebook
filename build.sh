#!/usr/binb/env bash

VERSION=$(git describe --tags)
build_aen() {
    if [ $GOOS == "windows" ]; then
      go build -o "dist/aen-$GOOS.exe" -ldflags "-X main.Version=$VERSION" -trimpath ./cmd/aen.go
    else
      go build -o "dist/aen-$GOOS" -ldflags "-X main.Version=$VERSION" -trimpath ./cmd/aen.go
    fi
}

export CGO_ENABLED=0
GOOS=windows build_aen
GOOS=linux build_aen
GOOS=darwin build_aen
