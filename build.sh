#!/usr/binb/env bash

VERSION=$(git describe --tags --abbrev=0)
build_aen() {
    if [ $GOOS == "windows" ]; then
      go build -o "dist/aen.exe" -ldflags "-X main.Version=$VERSION" -trimpath ./cmd/aen.go
      cd dist && 7z a aen-$VERSION-$GOOS-$GOARCH.zip aen.exe && cd ..
      rm dist/aen.exe
    else
      go build -o "dist/aen" -ldflags "-X main.Version=$VERSION" -trimpath ./cmd/aen.go
      cd dist && tar -cvzf aen-$VERSION-$GOOS-$GOARCH.tar.gz aen && cd ..
      rm dist/aen
    fi
}

export CGO_ENABLED=0
GOOS=windows GOARCH=amd64 build_aen
GOOS=linux GOARCH=amd64 build_aen
GOOS=darwin GOARCH=amd64 build_aen
