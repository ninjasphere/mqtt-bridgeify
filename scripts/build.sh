#!/usr/bin/env bash
set -e
set -x

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

PAHO_PATH=.gopath/src/git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git

if [ ! -d $PAHO_PATH ]; then
	git clone http://git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git $PAHO_PATH
	(cd $PAHO_PATH && git checkout 0d6c6e73b249ca8d48fde878b4d1cfbb4cd45a5e)
fi

export GOPATH="$(pwd)/.gopath"

# move the working path and build
cd .gopath/src/github.com/${OWNER}/${PROJECT_NAME}
go get -d -v ./...
go build -ldflags "-X main.GitCommit ${GIT_COMMIT}${GIT_DIRTY}" -o ${BIN_NAME}
mv ${BIN_NAME} ./bin
