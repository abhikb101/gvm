#!/bin/sh
set -e

VERSION=${1:-"dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

echo "Building GVM ${VERSION} (${COMMIT})..."

# Build for current platform
CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o gvm .
echo "Built: ./gvm"

# Cross-compile if --all flag is passed
if [ "$2" = "--all" ]; then
    mkdir -p dist

    for OS in darwin linux; do
        for ARCH in amd64 arm64; do
            OUTPUT="dist/gvm_${OS}_${ARCH}"
            echo "Building ${OS}/${ARCH}..."
            CGO_ENABLED=0 GOOS=${OS} GOARCH=${ARCH} go build -ldflags "${LDFLAGS}" -o "${OUTPUT}/gvm" .
            tar -czf "dist/gvm_${OS}_${ARCH}.tar.gz" -C "${OUTPUT}" gvm
            rm -rf "${OUTPUT}"
        done
    done

    echo "All builds in dist/"
fi
