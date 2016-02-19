package schema

import (
	// "github.com/aws/aws-sdk-go/aws"
	"github.com/graphql-go/graphql"
	// autoscaling "github.com/opsee/basic/schema/aws/autoscaling"
	ec2 "github.com/opsee/basic/schema/aws/ec2"
	// elb "github.com/opsee/basic/schema/aws/elb"
	rds "github.com/opsee/basic/schema/aws/rds"
	// googleproto "github.com/opsee/protobuf/proto/google/protobuf"
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestAWSSchema(t *testing.T) {
	popr := rand.New(rand.NewSource(time.Now().UnixNano()))

	instanceEC2 := &Instance{
		Id:         "i-666",
		CustomerId: "666-beef",
		Type:       "ec2",
		Resource: &Instance_Instance{
			Instance: ec2.NewPopulatedInstance(popr, false),
		},
	}

	instanceRDS := &Instance{
		Id:         "db-666",
		CustomerId: "666-beef",
		Type:       "rds",
		Resource: &Instance_DbInstance{
			DbInstance: rds.NewPopulatedDBInstance(popr, false),
		},
	}

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"instance_ec2": &graphql.Field{
					Type: GraphQLInstanceType,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return instanceEC2, nil
					},
				},
				"instance_rds": &graphql.Field{
					Type: GraphQLInstanceType,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return instanceRDS, nil
					},
				},
			},
		}),
	})

	if err != nil {
		t.Fatal(err)
	}

	queryResponse := graphql.Do(graphql.Params{Schema: schema, RequestString: `query yeOldeQuery {
		instance_ec2 {
			id
			customer_id
			type
			resource {
				... on ec2Instance {
					InstanceId
					InstanceType
					PrivateIpAddress
					State {
						Name
					}
					Tags {
						Key
						Value
					}
				}
			}
		}
		instance_rds {
			id
			customer_id
			type
			resource {
				... on rdsDBInstance {
					DBInstanceIdentifier
					DBName
					MultiAZ
					Endpoint {
						Address
						Port
					}
				}
			}
		}
	}`})

	if queryResponse.HasErrors() {
		t.Fatalf("graphql query errors: %#v\n", queryResponse.Errors)
	}

	instanceEC2ResponseJson, err := json.Marshal(getProp(queryResponse.Data, "instance_ec2", "resource"))
	if err != nil {
		t.Fatal(err)
	}
	instanceRDSResponseJson, err := json.Marshal(getProp(queryResponse.Data, "instance_rds", "resource"))
	if err != nil {
		t.Fatal(err)
	}

	decodedInstanceEC2 := &ec2.Instance{}
	err = json.NewDecoder(bytes.NewBuffer(instanceEC2ResponseJson)).Decode(decodedInstanceEC2)
	if err != nil {
		t.Fatal(err)
	}
	decodedInstanceRDS := &rds.DBInstance{}
	err = json.NewDecoder(bytes.NewBuffer(instanceRDSResponseJson)).Decode(decodedInstanceRDS)
	if err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, instanceEC2.Resource.(*Instance_Instance).Instance.InstanceId, decodedInstanceEC2.InstanceId)
	assert.EqualValues(t, instanceRDS.Resource.(*Instance_DbInstance).DbInstance.DBInstanceIdentifier, decodedInstanceRDS.DBInstanceIdentifier)
}

func getProp(i interface{}, path ...interface{}) interface{} {
	cur := i

	for _, s := range path {
		switch cur.(type) {
		case map[string]interface{}:
			cur = cur.(map[string]interface{})[s.(string)]
			continue
		case []interface{}:
			cur = cur.([]interface{})[s.(int)]
			continue
		default:
			return cur
		}
	}

	return cur
}
