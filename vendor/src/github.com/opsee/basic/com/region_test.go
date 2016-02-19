package com

import (
	"sort"
	"testing"
)

func TestSubnetSort(t *testing.T) {
	r1 := &Subnet{
		InstanceCount: 4,
		Routing:       RoutingStateNAT,
	}
	r2 := &Subnet{
		InstanceCount: 3,
		Routing:       RoutingStateNAT,
	}
	r3 := &Subnet{
		InstanceCount: 3,
		Routing:       RoutingStateGateway,
	}
	r4 := &Subnet{
		InstanceCount: 2,
		Routing:       RoutingStateGateway,
	}

	subnetz := []*Subnet{r4, r1, r3, r2}
	sort.Sort(SubnetsByPreference(subnetz))

	expected := []*Subnet{r1, r2, r3, r4}
	for i, s := range expected {
		if subnetz[i] != s {
			t.Fatal("subnets not sorted correctly")
		}
	}
}
