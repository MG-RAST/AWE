#!/bin/sh
set -e


# append "-a" for all


touch lib/conf/conf.go
set -x
CGO_ENABLED=0 go install $1 -installsuffix cgo -v -ldflags="-X github.com/MG-RAST/AWE/lib/conf.VERSION=$(git describe --tags --long)" ./awe-submitter/
set +x

