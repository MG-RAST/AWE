#!/bin/sh
set -e


# append "-a" for all


AWE_VERSION=$(git describe --tags --long)

touch lib/conf/conf.go
set -x
CGO_ENABLED=0 go install $1 -installsuffix cgo -v -ldflags="-X github.com/MG-RAST/AWE/lib/conf.VERSION=${AWE_VERSION}" ./awe-server/
set +x


# use this for race debug flag
#go install -a -v -race -ldflags "-X github.com/MG-RAST/AWE/lib/conf.GIT_COMMIT_HASH=${GITHASH}" ./awe-worker/ ./awe-server/
