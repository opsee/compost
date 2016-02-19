#!/bin/bash

GOPATH="$GOPATH:/build"
GO15VENDOREXPERIMENT=1 
proto_dir=./schema

go get ./cmd/... && go build ./cmd/...

mkdir -p ./schema/aws
awsproto -basepath "$PWD/schema/aws"
