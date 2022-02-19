#!/usr/binb/env bash

VERSION=$(git describe --tags --abbrev=0)
build_aen() {
    echo Building $GOOS $GOARCH...
    if [ $GOOS == "windows" ]; then
      go build -o "dist/aen.exe" -ldflags "-X main.Version=$VERSION" -trimpath ./cmd/aen.go > /dev/null
      cd dist && 7z a aen-$VERSION-$GOOS-$GOARCH.zip aen.exe > /dev/null 2>&1 && cd ..
      rm dist/aen.exe
    else
      go build -o "dist/aen" -ldflags "-X main.Version=$VERSION" -trimpath ./cmd/aen.go > /dev/null
      cd dist && tar -cvzf aen-$VERSION-$GOOS-$GOARCH.tar.gz aen > /dev/null 2>&1 && cd ..
      rm dist/aen
    fi
}

export CGO_ENABLED=0
GOOS=windows GOARCH=amd64 build_aen
GOOS=linux GOARCH=amd64 build_aen
GOOS=darwin GOARCH=amd64 build_aen
GOOS=windows GOARCH=arm64 build_aen
GOOS=linux GOARCH=arm64 build_aen
GOOS=darwin GOARCH=arm64 build_aen
