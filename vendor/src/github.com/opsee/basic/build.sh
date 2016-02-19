#!/bin/bash
set -e

GOPATH="$GOPATH:/build"
GO15VENDOREXPERIMENT=1 
proto_dir=./schema

for d in ${proto_dir}/**/**/; do
  protoc --gogoopsee_out=plugins=grpc+graphql,Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:${d} --proto_path=/gopath/src:${d} ${d}/*.proto
done

protoc --gogoopsee_out=plugins=grpc+graphql,Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:${proto_dir} --proto_path=/gopath/src:${proto_dir} ${proto_dir}/*.proto

go get -t ./... && \
  go test -v ./...
