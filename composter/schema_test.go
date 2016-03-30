package composter

import (
	// "github.com/graphql-go/graphql"
	// "github.com/stretchr/testify/assert"
	// "encoding/json"
	"testing"
)

func TestSchema(t *testing.T) {
	_ = `query groupQuery {
		group(id: "sg-badbadbad", type: "security") {
			id
			type
			instance_count
			
			resource {
				... on aws.autoscaling.Group {
					AutoScalingGroupName
					Status
					TagDescription {
						Key
						Value
					}
				}
				
				... on aws.ec2.SecurityGroup {
					GroupName
					GroupId
					Description
					Tags {
						Key
						Value
					}
				}
				
				... on aws.elb.LoadBalancerDescription {
					DNSName
					LoadBalancerName
					CanonicalHostedZoneName
				}
			}
			
			checks {
				id
				interval
				target {
					name
					id
					type
					address
				}
				name
				assertions {
					key
					value
					relationship
					operand
				}
			}
			
			instances {
				id
				type
				
				result {
					passing
					responses {
						response
						error
						passing
					}
				}
			}
		}
	}`
}

// {"checks":[{"id":"up8ZQoHRDYbJL8mSS3z8Y","interval":30,"check_spec":{"value":{"name":"WELCOME","path":"/","port":80,"verb":"GET","protocol":"http"},"type_url":"HttpCheck"},"last_run":null,"name":"WELCOME","assertions":[{"check_id":"up8ZQoHRDYbJL8mSS3z8Y","customer_id":"a1de53d8-8974-11e5-9e7b-f349fb6fa040","key":"body","relationship":"contain","value":"","operand":"Welcome"}],"target":{"name":"test group","type":"sg","id":"sg-c6551ca2"}},{"id":"72u23sJlP3ZBjUWYlPWZx5","interval":30,"check_spec":{"value":{"name":"Http test group","path":"/","port":80,"verb":"GET","protocol":"http"},"type_url":"HttpCheck"},"last_run":null,"name":"Http test group","assertions":[{"check_id":"72u23sJlP3ZBjUWYlPWZx5","customer_id":"a1de53d8-8974-11e5-9e7b-f349fb6fa040","key":"code","relationship":"equal","value":"","operand":"200"}],"target":{"name":"test group","type":"sg","id":"sg-c6551ca2"}}]}
