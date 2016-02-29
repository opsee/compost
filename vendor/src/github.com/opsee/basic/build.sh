#!/bin/bash
set -e

GOPATH="$GOPATH:/build"
GO15VENDOREXPERIMENT=1 
proto_dir=./schema
grpc_dir=./service

for d in ${proto_dir}/**/**/; do
  protoc --gogoopsee_out=plugins=grpc+graphql,Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:${d} --proto_path=/gopath/src:${d} ${d}/*.proto
done

protoc --gogoopsee_out=plugins=grpc+graphql,Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:${proto_dir} --proto_path=/gopath/src:${proto_dir} ${proto_dir}/*.proto
protoc --gogoopsee_out=plugins=grpc+graphql,Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor,Mstack.proto=github.com/opsee/basic/schema:${grpc_dir} --proto_path=/gopath/src:${proto_dir}:${grpc_dir} ${grpc_dir}/*.proto

go get -u google.golang.org/grpc
go get -t ./... && \
  go test -v ./...
