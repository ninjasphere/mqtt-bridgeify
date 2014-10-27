#!/usr/bin/env bash
set -ex

OWNER=ninjablocks
BIN_NAME=mqtt-bridgeify
PROJECT_NAME=mqtt-bridgeify

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

GIT_COMMIT="$(git rev-parse HEAD)"
GIT_DIRTY="$(test -n "`git status --porcelain`" && echo "+CHANGES" || true)"
VERSION="$(grep "const Version " version.go | sed -E 's/.*"(.+)"$/\1/' )"

# remove working build
# rm -rf .gopath
if [ ! -d ".gopath" ]; then
	mkdir -p .gopath/src/github.com/${OWNER}
	ln -sf ../../../.. .gopath/src/github.com/${OWNER}/${PROJECT_NAME}
fi

export GOPATH="$(pwd)/.gopath"

if [ ! -d "$GOPATH/src/github.com/wolfeidau/org.eclipse.paho.mqtt.golang" ]; then
	# Clone my fork of paho client
	git clone -b develop https://github.com/wolfeidau/org.eclipse.paho.mqtt.golang $GOPATH/src/github.com/wolfeidau/org.eclipse.paho.mqtt.golang
fi

# move the working path and build
cd .gopath/src/github.com/${OWNER}/${PROJECT_NAME}
go get -d -v ./...
go build -ldflags "-X main.GitCommit ${GIT_COMMIT}${GIT_DIRTY}" -o bin/${BIN_NAME}
