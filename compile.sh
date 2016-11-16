#!/bin/sh
set -e
set -x

AWE="."

GITHASH=$(git -C ${AWE} rev-parse HEAD)
CGO_ENABLED=0 go install -a -installsuffix cgo -v -ldflags "-X github.com/MG-RAST/AWE/lib/conf.GIT_COMMIT_HASH=${GITHASH}" ...
