package com

import (
	"github.com/aws/aws-sdk-go/service/elb"
	"reflect"
)

var (
	AWSTypeFactory = make(map[string]reflect.Type)
)

func init() {
	AWSTypeFactory[reflect.TypeOf(elb.LoadBalancerDescription{}).Name()] = reflect.TypeOf(elb.LoadBalancerDescription{})
}

type AWSObject struct {
	Type   string
	Object interface{}
	Owner  *User
}
