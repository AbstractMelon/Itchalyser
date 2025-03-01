#!/bin/bash

set -e

# Function to build for a specific OS and architecture
build() {
    local os=$1
    local arch=$2
    local output="../builds/$os/Itchalyser"

    mkdir -p $(dirname $output)
    echo "Building for $os-$arch"
    cd src
    GOOS=$os GOARCH=$arch go build -o $output main.go
    echo "Built $output"
    cd ..
}

# Usage:
# ./build.sh <os> <arch>
# Example: ./build.sh linux amd64

if [ "$1" = "all" ]; then
    build windows amd64
    build linux amd64
    build android arm64
else
    if [ -z "$1" -o -z "$2" ]; then
        echo "Usage: ./build.sh <os> <arch>"
        echo "Example: ./build.sh linux amd64"
        exit 1
    fi

    build $1 $2
fi

