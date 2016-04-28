package schema

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
