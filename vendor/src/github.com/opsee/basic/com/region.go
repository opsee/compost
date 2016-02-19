package com

const (
	// subnet has no route to the internet
	RoutingStatePrivate = "private"
	// subnet has a route to the internet via an aws internet gateway
	RoutingStatePublic = "public"
	// subnet has a route to the internet via a NAT instance
	RoutingStateNAT = "nat"
	// subnet has a route to the internet via a customer gateway
	RoutingStateGateway = "gateway"
	// subnet may have a route to the internet, but can't communicate with
	// 100% of instances in a VPC
	RoutingStateOccluded = "occluded"
)

var (
	RoutingPreference = map[string]int{
		RoutingStateNAT:      0,
		RoutingStateGateway:  1,
		RoutingStatePublic:   2,
		RoutingStateOccluded: 3,
		RoutingStatePrivate:  4,
	}
)

type Region struct {
	CustomerID         string    `json:"-" db:"customer_id"`
	Region             string    `json:"region"`
	SupportedPlatforms []*string `json:"supported_platforms"`
	VPCs               []*VPC    `json:"vpcs"`
	Subnets            []*Subnet `json:"subnets"`
}

type VPC struct {
	CidrBlock       *string `json:"cidr_block"`
	DhcpOptionsId   *string `json:"dhcp_options_id"`
	InstanceTenancy *string `json:"instance_tenancy"`
	IsDefault       *bool   `json:"is_default"`
	State           *string `json:"state"`
	VpcId           *string `json:"vpc_id"`
	Tags            []*Tag  `json:"tags"`
	InstanceCount   int     `json:"instance_count"`
}

type Subnet struct {
	AvailabilityZone        *string `json:"availability_zone"`
	AvailableIpAddressCount *int64  `json:"available_ip_address_count"`
	CidrBlock               *string `json:"cidr_block"`
	DefaultForAz            *bool   `json:"default_for_az"`
	MapPublicIpOnLaunch     *bool   `json:"map_public_ip_on_launch"`
	State                   *string `json:"state"`
	SubnetId                *string `json:"subnet_id"`
	VpcId                   *string `json:"vpc_id"`
	Tags                    []*Tag  `json:"tags"`
	InstanceCount           int     `json:"instance_count"`
	Routing                 string  `json:"routing"`
}

type Tag struct {
	Key   *string `json:"key"`
	Value *string `json:"value"`
}

type SubnetsByPreference []*Subnet

func (s SubnetsByPreference) Len() int      { return len(s) }
func (s SubnetsByPreference) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SubnetsByPreference) Less(i, j int) bool {
	l, r := RoutingPreference[s[i].Routing], RoutingPreference[s[j].Routing]
	if l == r {
		return s[i].InstanceCount > s[j].InstanceCount
	}

	return l < r
}
