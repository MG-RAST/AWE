#!/bin/sh
cd $GOPATH

echo "####### go fmt AWE     #######"
go fmt github.com/MG-RAST/AWE/...

echo "####### go fix AWE     #######"
go fix github.com/MG-RAST/AWE/...

echo "####### go install AWE #######"
go install -v github.com/MG-RAST/AWE/...

