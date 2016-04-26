package composter

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/graphql-go/graphql"
	"github.com/opsee/basic/schema"
	opsee_aws_autoscaling "github.com/opsee/basic/schema/aws/autoscaling"
	opsee_aws_cloudwatch "github.com/opsee/basic/schema/aws/cloudwatch"
	opsee_aws_ec2 "github.com/opsee/basic/schema/aws/ec2"
	opsee_aws_elb "github.com/opsee/basic/schema/aws/elb"
	opsee_aws_rds "github.com/opsee/basic/schema/aws/rds"
	opsee "github.com/opsee/basic/service"
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
	"time"
	// log "github.com/sirupsen/logrus"
)

var (
	errDecodeUser                  = errors.New("error decoding user")
	errDecodeQueryContext          = errors.New("error decoding query context")
	errMissingRegion               = errors.New("missing region id")
	errMissingVpc                  = errors.New("missing vpc id")
	errMissingInstanceType         = errors.New("missing instance type - must be one of (ec2, rds)")
	errMissingGroupType            = errors.New("missing group type - must be one of (security, elb, autoscaling)")
	errDecodeInstances             = errors.New("error decoding instances")
	errDecodeInstanceIds           = errors.New("error decoding instance ids")
	errUnknownInstanceMetricType   = errors.New("no metrics for that instance type")
	errDecodeMetricStatisticsInput = errors.New("error decoding metric statistics input")
	errDecodeCheckInput            = errors.New("error decoding checks input")
	errUnknownAction               = errors.New("unknown action")

	InstanceType   *graphql.Object
	DbInstanceType *graphql.Object
	CheckType      *graphql.Object
	CheckInputType *graphql.InputObject
)

type instanceAction int

const (
	instanceReboot instanceAction = iota
	instanceStart
	instanceStop
)

func (c *Composter) mustSchema() {
	c.initTypes()

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    c.query(),
		Mutation: c.mutation(),
	})

	if err != nil {
		panic(fmt.Sprint("error generating graphql schema: ", err))
	}

	adminSchema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    c.adminQuery(),
		Mutation: c.mutation(),
	})

	if err != nil {
		panic(fmt.Sprint("error generating graphql schema: ", err))
	}

	c.Schema = schema
	c.AdminSchema = adminSchema
}

func (c *Composter) initTypes() {
	metrics := c.queryMetrics()

	if InstanceType == nil {
		InstanceType = graphql.NewObject(graphql.ObjectConfig{
			Name: opsee_aws_ec2.GraphQLInstanceType.Name(),
			Fields: graphql.Fields{
				"metrics": metrics,
			},
		})
		addFields(InstanceType, opsee_aws_ec2.GraphQLInstanceType.Fields())
	}

	if DbInstanceType == nil {
		DbInstanceType = graphql.NewObject(graphql.ObjectConfig{
			Name: opsee_aws_rds.GraphQLDBInstanceType.Name(),
			Fields: graphql.Fields{
				"metrics": metrics,
			},
		})
		addFields(DbInstanceType, opsee_aws_rds.GraphQLDBInstanceType.Fields())
	}

	if CheckType == nil {
		CheckType = graphql.NewObject(graphql.ObjectConfig{
			Name: schema.GraphQLCheckType.Name(),
			Fields: graphql.Fields{
				"notifications": &graphql.Field{
					Type: graphql.NewList(schema.GraphQLNotificationType),
				},
			},
		})
		addFields(CheckType, schema.GraphQLCheckType.Fields())
	}

	if CheckInputType == nil {
		CheckInputType = graphql.NewInputObject(graphql.InputObjectConfig{
			Name:        "Check",
			Description: "An Opsee Check",
			Fields: graphql.InputObjectConfigFieldMap{
				"name": &graphql.InputObjectFieldConfig{
					Type:        graphql.NewNonNull(graphql.String),
					Description: "The check name",
				},
				"http_check": &graphql.InputObjectFieldConfig{
					Type: graphql.NewInputObject(graphql.InputObjectConfig{
						Name:        "HTTPCheck",
						Description: "checks ur http",
						Fields: graphql.InputObjectConfigFieldMap{
							"path": &graphql.InputObjectFieldConfig{
								Type:        graphql.NewNonNull(graphql.String),
								Description: "The path to check",
							},
							"protocol": &graphql.InputObjectFieldConfig{
								Type:        graphql.NewNonNull(graphql.String),
								Description: "The protocol to check",
							},
							"port": &graphql.InputObjectFieldConfig{
								Type:        graphql.NewNonNull(graphql.Int),
								Description: "The port to check",
							},
							"verb": &graphql.InputObjectFieldConfig{
								Type:        graphql.NewNonNull(graphql.String),
								Description: "The verb to check",
							},
							"headers": &graphql.InputObjectFieldConfig{
								Type: graphql.NewList(graphql.NewInputObject(graphql.InputObjectConfig{
									Name:        "Header",
									Description: "HTTP Header",
									Fields: graphql.InputObjectConfigFieldMap{
										"name": &graphql.InputObjectFieldConfig{
											Type:        graphql.NewNonNull(graphql.String),
											Description: "Header name",
										},
										"values": &graphql.InputObjectFieldConfig{
											Type:        graphql.NewList(graphql.String),
											Description: "Header values",
										},
									},
								})),
								Description: "Headers to send",
							},
							"body": &graphql.InputObjectFieldConfig{
								Type:        graphql.String,
								Description: "A request body to send",
							},
						},
					}),
					Description: "An HTTP check",
				},
				"cloudwatch_check": &graphql.InputObjectFieldConfig{
					Type: graphql.NewInputObject(graphql.InputObjectConfig{
						Name:        "CloudwatchCheck",
						Description: "checks ur cloudwatch metrics",
						Fields: graphql.InputObjectConfigFieldMap{
							"metrics": &graphql.InputObjectFieldConfig{
								Type: graphql.NewList(graphql.NewInputObject(graphql.InputObjectConfig{
									Name:        "Metric",
									Description: "A cloudwatch metric source",
									Fields: graphql.InputObjectConfigFieldMap{
										"namespace": &graphql.InputObjectFieldConfig{
											Type:        graphql.NewNonNull(graphql.String),
											Description: "The cloudwatch metric namespace",
										},
										"name": &graphql.InputObjectFieldConfig{
											Type:        graphql.NewNonNull(graphql.String),
											Description: "The cloudwatch metric name",
										},
									},
								})),
							},
						},
					}),
					Description: "A cloudwatch check",
				},
				"target": &graphql.InputObjectFieldConfig{
					Type: graphql.NewNonNull(graphql.NewInputObject(graphql.InputObjectConfig{
						Name:        "Target",
						Description: "An AWS resource to target",
						Fields: graphql.InputObjectConfigFieldMap{
							"name": &graphql.InputObjectFieldConfig{
								Type:        graphql.String,
								Description: "The target name",
							},
							"type": &graphql.InputObjectFieldConfig{
								Type:        graphql.NewNonNull(graphql.String),
								Description: "The target type",
							},
							"id": &graphql.InputObjectFieldConfig{
								Type:        graphql.NewNonNull(graphql.String),
								Description: "The target id",
							},
						},
					})),
					Description: "A check target",
				},
				"assertions": &graphql.InputObjectFieldConfig{
					Type: graphql.NewNonNull(graphql.NewList(graphql.NewInputObject(graphql.InputObjectConfig{
						Name:        "Assertion",
						Description: "An assertion to apply to a check target",
						Fields: graphql.InputObjectConfigFieldMap{
							"key": &graphql.InputObjectFieldConfig{
								Type:        graphql.String,
								Description: "[TODO]",
							},
							"value": &graphql.InputObjectFieldConfig{
								Type:        graphql.String,
								Description: "[TODO]",
							},
							"relationship": &graphql.InputObjectFieldConfig{
								Type:        graphql.NewNonNull(graphql.String),
								Description: "[TODO]",
							},
							"operand": &graphql.InputObjectFieldConfig{
								Type:        graphql.String,
								Description: "[TODO]",
							},
						},
					}))),
					Description: "Check assertions",
				},
				"notifications": &graphql.InputObjectFieldConfig{
					Type: graphql.NewNonNull(graphql.NewList(graphql.NewInputObject(graphql.InputObjectConfig{
						Name:        "Notification",
						Description: "A notification endpoint for failing / passing checks",
						Fields: graphql.InputObjectConfigFieldMap{
							"type": &graphql.InputObjectFieldConfig{
								Type:        graphql.NewNonNull(graphql.String),
								Description: "A notification type, such as slack_bot, email",
							},
							"value": &graphql.InputObjectFieldConfig{
								Type:        graphql.NewNonNull(graphql.String),
								Description: "A notification value, such as an email address or slack channel",
							},
						},
					}))),
					Description: "Check notifications",
				},
			},
		})
	}
}

func (c *Composter) query() *graphql.Object {
	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"checks": c.queryChecks(),
			"region": c.queryRegion(),
		},
	})

	return query
}

func (c *Composter) adminQuery() *graphql.Object {
	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"checks": c.queryChecks(),
			"region": c.queryRegion(),
			"listCustomers": &graphql.Field{
				Type: opsee.GraphQLListCustomersResponseType,
				Args: graphql.FieldConfigArgument{
					"page": &graphql.ArgumentConfig{
						Description: "The page number.",
						Type:        graphql.Int,
					},
					"per_page": &graphql.ArgumentConfig{
						Description: "The number of customers per page.",
						Type:        graphql.Int,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					_, ok := p.Context.Value(userKey).(*schema.User)
					if !ok {
						return nil, errDecodeUser
					}

					var (
						page    int
						perPage int
					)

					page, ok = p.Args["page"].(int)
					perPage, ok = p.Args["per_page"].(int)

					return c.resolver.ListCustomers(p.Context, &opsee.ListUsersRequest{
						Page:    int32(page),
						PerPage: int32(perPage),
					})
				},
			},
			"getUser": &graphql.Field{
				Type: opsee.GraphQLGetUserResponseType,
				Args: graphql.FieldConfigArgument{
					"customer_id": &graphql.ArgumentConfig{
						Description: "The customer Id.",
						Type:        graphql.String,
					},
					"email": &graphql.ArgumentConfig{
						Description: "The user's email.",
						Type:        graphql.String,
					},
					"id": &graphql.ArgumentConfig{
						Description: "The user's id.",
						Type:        graphql.Int,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					_, ok := p.Context.Value(userKey).(*schema.User)
					if !ok {
						return nil, errDecodeUser
					}

					var (
						customerId string
						email      string
						id         int
					)

					customerId, ok = p.Args["customer_id"].(string)
					email, ok = p.Args["email"].(string)
					id, ok = p.Args["id"].(int)

					return c.resolver.GetUser(p.Context, &opsee.GetUserRequest{
						CustomerId: customerId,
						Email:      email,
						Id:         int32(id),
					})
				},
			},
			"getCredentials": &graphql.Field{
				Type: opsee.GraphQLGetCredentialsResponseType,
				Args: graphql.FieldConfigArgument{
					"customer_id": &graphql.ArgumentConfig{
						Description: "The customer Id.",
						Type:        graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					_, ok := p.Context.Value(userKey).(*schema.User)
					if !ok {
						return nil, errDecodeUser
					}

					var (
						customerId string
					)

					customerId, ok = p.Args["customer_id"].(string)

					return c.resolver.GetCredentials(p.Context, customerId)
				},
			},
		},
	})

	return query
}

func (c *Composter) queryChecks() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(CheckType),
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "A single check Id",
				Type:        graphql.String,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			id, _ := p.Args["id"].(string)

			return c.resolver.ListChecks(p.Context, user, id)
		},
	}
}

func (c *Composter) queryRegion() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewObject(graphql.ObjectConfig{
			Name:        "Region",
			Description: "The AWS Region",
			Fields: graphql.Fields{
				"vpc": c.queryVpc(),
			},
		}),
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "The region id",
				Type:        graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			_, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			queryContext, ok := p.Context.Value(queryContextKey).(*QueryContext)
			if !ok {
				return nil, errDecodeQueryContext
			}

			region, _ := p.Args["id"].(string)
			if region == "" {
				return nil, errMissingRegion
			}

			queryContext.Region = region

			return struct{}{}, nil
		},
	}
}

func (c *Composter) queryVpc() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewObject(graphql.ObjectConfig{
			Name:        "VPC",
			Description: "An AWS VPC",
			Fields: graphql.Fields{
				"groups":          c.queryGroups(),
				"instances":       c.queryInstances(),
				"rebootInstances": c.instanceAction(instanceReboot),
				"startInstances":  c.instanceAction(instanceStart),
				"stopInstances":   c.instanceAction(instanceStop),
			},
		}),
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "The VPC id",
				Type:        graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			_, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			queryContext, ok := p.Context.Value(queryContextKey).(*QueryContext)
			if !ok {
				return nil, errDecodeQueryContext
			}

			vpc, _ := p.Args["id"].(string)
			if vpc == "" {
				return nil, errMissingVpc
			}

			queryContext.VpcId = vpc

			return struct{}{}, nil
		},
	}
}

func (c *Composter) queryGroups() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(graphql.NewUnion(graphql.UnionConfig{
			Name:        "Group",
			Description: "A group target",
			Types: []*graphql.Object{
				opsee_aws_ec2.GraphQLSecurityGroupType,
				opsee_aws_elb.GraphQLLoadBalancerDescriptionType,
				opsee_aws_autoscaling.GraphQLGroupType,
			},
			ResolveType: func(value interface{}, info graphql.ResolveInfo) *graphql.Object {
				switch value.(type) {
				case *opsee_aws_ec2.SecurityGroup:
					return opsee_aws_ec2.GraphQLSecurityGroupType
				case *opsee_aws_elb.LoadBalancerDescription:
					return opsee_aws_elb.GraphQLLoadBalancerDescriptionType
				case *opsee_aws_autoscaling.Group:
					return opsee_aws_autoscaling.GraphQLGroupType
				}
				return nil
			},
		})),
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "An optional group identifier",
				Type:        graphql.String,
			},
			"type": &graphql.ArgumentConfig{
				Description: "A group type (security, elb, autoscaling)",
				Type:        graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			queryContext, ok := p.Context.Value(queryContextKey).(*QueryContext)
			if !ok {
				return nil, errDecodeQueryContext
			}

			groupId, _ := p.Args["id"].(string)
			groupType, _ := p.Args["type"].(string)

			if groupType == "" {
				return nil, errMissingGroupType
			}

			return c.resolver.GetGroups(p.Context, user, queryContext.Region, queryContext.VpcId, groupType, groupId)
		},
	}
}

func (c *Composter) queryInstances() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(graphql.NewUnion(graphql.UnionConfig{
			Name:        "Instance",
			Description: "An instance target",
			Types: []*graphql.Object{
				InstanceType,
				DbInstanceType,
			},
			ResolveType: func(value interface{}, info graphql.ResolveInfo) *graphql.Object {
				switch value.(type) {
				case *opsee_aws_ec2.Instance:
					return InstanceType
				case *opsee_aws_rds.DBInstance:
					return DbInstanceType
				}
				return nil
			},
		})),
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "An optional instance id",
				Type:        graphql.String,
			},
			"type": &graphql.ArgumentConfig{
				Description: "An instance type (rds, ec2)",
				Type:        graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			queryContext, ok := p.Context.Value(queryContextKey).(*QueryContext)
			if !ok {
				return nil, errDecodeQueryContext
			}

			instanceId, _ := p.Args["id"].(string)
			instanceType, _ := p.Args["type"].(string)

			if instanceType == "" {
				return nil, errMissingInstanceType
			}

			return c.resolver.GetInstances(p.Context, user, queryContext.Region, queryContext.VpcId, instanceType, instanceId)
		},
	}
}

func (c *Composter) instanceAction(action instanceAction) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(graphql.String),
		Args: graphql.FieldConfigArgument{
			"ids": &graphql.ArgumentConfig{
				Description: "A list of instance ids",
				Type:        graphql.NewNonNull(graphql.NewList(graphql.String)),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			queryContext, ok := p.Context.Value(queryContextKey).(*QueryContext)
			if !ok {
				return nil, errDecodeQueryContext
			}

			var ids []string
			idArgs, ok := p.Args["ids"].([]interface{})
			if !ok {
				return nil, errDecodeInstanceIds
			}

			for _, id := range idArgs {
				idstr, ok := id.(string)
				if !ok {
					return nil, errDecodeInstanceIds
				}

				ids = append(ids, idstr)
			}

			var err error

			switch action {
			case instanceReboot:
				err = c.resolver.RebootInstances(p.Context, user, queryContext.Region, ids)
			case instanceStart:
				err = c.resolver.StartInstances(p.Context, user, queryContext.Region, ids)
			case instanceStop:
				err = c.resolver.StopInstances(p.Context, user, queryContext.Region, ids)
			default:
				err = errUnknownAction
			}

			if err != nil {
				return nil, err
			}

			return ids, nil
		},
	}
}

func (c *Composter) queryMetrics() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewObject(graphql.ObjectConfig{
			Name:        "Metrics",
			Description: "Cloudwatch instance metrics",
			Fields: graphql.Fields{
				"BackendConnectionErrors":               c.queryMetricName("BackendConnectionErrors"),
				"CPUCreditUsage":                        c.queryMetricName("CPUCreditUsage"),
				"CPUCreditBalance":                      c.queryMetricName("CPUCreditBalance"),
				"VolumeWriteBytes":                      c.queryMetricName("VolumeWriteBytes"),
				"VolumeIdleTime":                        c.queryMetricName("VolumeIdleTime"),
				"StatusCheckFailed_Instance":            c.queryMetricName("StatusCheckFailed_Instance"),
				"StatusCheckFailed_System":              c.queryMetricName("StatusCheckFailed_System"),
				"StatusCheckFailed":                     c.queryMetricName("StatusCheckFailed"),
				"VolumeReadBytes":                       c.queryMetricName("VolumeReadBytes"),
				"SurgeQueueLength":                      c.queryMetricName("SurgeQueueLength"),
				"ReadLatency":                           c.queryMetricName("ReadLatency"),
				"DiskWriteOps":                          c.queryMetricName("DiskWriteOps"),
				"NetworkPacketsOut":                     c.queryMetricName("NetworkPacketsOut"),
				"OldestReplicationSlotLag":              c.queryMetricName("OldestReplicationSlotLag"),
				"RequestCount":                          c.queryMetricName("RequestCount"),
				"NumberOfMessagesPublished":             c.queryMetricName("NumberOfMessagesPublished"),
				"NumberOfEmptyReceives":                 c.queryMetricName("NumberOfEmptyReceives"),
				"Evictions":                             c.queryMetricName("Evictions"),
				"CurrItems":                             c.queryMetricName("CurrItems"),
				"CurrConnections":                       c.queryMetricName("CurrConnections"),
				"HealthyHostCount":                      c.queryMetricName("HealthyHostCount"),
				"CmdTouch":                              c.queryMetricName("CmdTouch"),
				"UnHealthyHostCount":                    c.queryMetricName("UnHealthyHostCount"),
				"ReadIOPS":                              c.queryMetricName("ReadIOPS"),
				"ReadThroughput":                        c.queryMetricName("ReadThroughput"),
				"CasHits":                               c.queryMetricName("CasHits"),
				"VolumeQueueLength":                     c.queryMetricName("VolumeQueueLength"),
				"SwapUsage":                             c.queryMetricName("SwapUsage"),
				"MemoryReservation":                     c.queryMetricName("MemoryReservation"),
				"NetworkReceiveThroughput":              c.queryMetricName("NetworkReceiveThroughput"),
				"GetMisses":                             c.queryMetricName("GetMisses"),
				"ApproximateNumberOfMessagesNotVisible": c.queryMetricName("ApproximateNumberOfMessagesNotVisible"),
				"CPUUtilization":                        c.queryMetricName("CPUUtilization"),
				"DatabaseConnections":                   c.queryMetricName("DatabaseConnections"),
				"Latency":                               c.queryMetricName("Latency"),
				"VolumeWriteOps":                        c.queryMetricName("VolumeWriteOps"),
				"MemoryUtilization":                     c.queryMetricName("MemoryUtilization"),
				"FreeStorageSpace":                      c.queryMetricName("FreeStorageSpace"),
				"EvictedUnfetched":                      c.queryMetricName("EvictedUnfetched"),
				"DecrMisses":                            c.queryMetricName("DecrMisses"),
				"CmdGet":                                c.queryMetricName("CmdGet"),
				"HTTPCode_Backend_2XX":                  c.queryMetricName("HTTPCode_Backend_2XX"),
				"HTTPCode_Backend_3XX":                  c.queryMetricName("HTTPCode_Backend_3XX"),
				"DiskWriteBytes":                        c.queryMetricName("DiskWriteBytes"),
				"TransactionLogsDiskUsage":              c.queryMetricName("TransactionLogsDiskUsage"),
				"DiskReadBytes":                         c.queryMetricName("DiskReadBytes"),
				"DiskReadOps":                           c.queryMetricName("DiskReadOps"),
				"NetworkIn":                             c.queryMetricName("NetworkIn"),
				"ApproximateNumberOfMessagesVisible":    c.queryMetricName("ApproximateNumberOfMessagesVisible"),
				"NumberOfMessagesSent":                  c.queryMetricName("NumberOfMessagesSent"),
				"WriteIOPS":                             c.queryMetricName("WriteIOPS"),
				"HTTPCode_Backend_5XX":                  c.queryMetricName("HTTPCode_Backend_5XX"),
				"BytesUsedForHash":                      c.queryMetricName("BytesUsedForHash"),
				"NewConnections":                        c.queryMetricName("NewConnections"),
				"DiskQueueDepth":                        c.queryMetricName("DiskQueueDepth"),
				"WriteLatency":                          c.queryMetricName("WriteLatency"),
				"DeleteHits":                            c.queryMetricName("DeleteHits"),
				"NetworkTransmitThroughput":             c.queryMetricName("NetworkTransmitThroughput"),
				"BinLogDiskUsage":                       c.queryMetricName("BinLogDiskUsage"),
				"VolumeTotalWriteTime":                  c.queryMetricName("VolumeTotalWriteTime"),
				"TouchMisses":                           c.queryMetricName("TouchMisses"),
				"CmdConfigGet":                          c.queryMetricName("CmdConfigGet"),
				"IncrMisses":                            c.queryMetricName("IncrMisses"),
				"FreeableMemory":                        c.queryMetricName("FreeableMemory"),
				"Throttles":                             c.queryMetricName("Throttles"),
				"Reclaimed":                             c.queryMetricName("Reclaimed"),
				"NetworkPacketsIn":                      c.queryMetricName("NetworkPacketsIn"),
				"IncrHits":                              c.queryMetricName("IncrHits"),
				"VolumeReadOps":                         c.queryMetricName("VolumeReadOps"),
				"WriteThroughput":                       c.queryMetricName("WriteThroughput"),
				"BucketSizeBytes":                       c.queryMetricName("BucketSizeBytes"),
				"NetworkOut":                            c.queryMetricName("NetworkOut"),
				"ReplicaLag":                            c.queryMetricName("ReplicaLag"),
				"TriggeredRules":                        c.queryMetricName("TriggeredRules"),
				"BytesUsedForCacheItems":                c.queryMetricName("BytesUsedForCacheItems"),
				"Invocations":                           c.queryMetricName("Invocations"),
				"IncomingLogEvents":                     c.queryMetricName("IncomingLogEvents"),
				"TouchHits":                             c.queryMetricName("TouchHits"),
				"DecrHits":                              c.queryMetricName("DecrHits"),
				"VolumeTotalReadTime":                   c.queryMetricName("VolumeTotalReadTime"),
				"NetworkBytesOut":                       c.queryMetricName("NetworkBytesOut"),
				"HTTPCode_Backend_4XX":                  c.queryMetricName("HTTPCode_Backend_4XX"),
				"CasMisses":                             c.queryMetricName("CasMisses"),
				"NumberOfNotificationsFailed":           c.queryMetricName("NumberOfNotificationsFailed"),
				"HTTPCode_ELB_5XX":                      c.queryMetricName("HTTPCode_ELB_5XX"),
				"ExpiredUnfetched":                      c.queryMetricName("ExpiredUnfetched"),
				"NumberOfObjects":                       c.queryMetricName("NumberOfObjects"),
				"NewItems":                              c.queryMetricName("NewItems"),
				"CurrConfig":                            c.queryMetricName("CurrConfig"),
				"CmdConfigSet":                          c.queryMetricName("CmdConfigSet"),
				"CmdFlush":                              c.queryMetricName("CmdFlush"),
				"SentMessageSize":                       c.queryMetricName("SentMessageSize"),
				"NumberOfMessagesDeleted":               c.queryMetricName("NumberOfMessagesDeleted"),
				"PublishSize":                           c.queryMetricName("PublishSize"),
				"NumberOfMessagesReceived":              c.queryMetricName("NumberOfMessagesReceived"),
				"UnusedMemory":                          c.queryMetricName("UnusedMemory"),
				"NumberOfNotificationsDelivered":        c.queryMetricName("NumberOfNotificationsDelivered"),
				"CmdSet":                 c.queryMetricName("CmdSet"),
				"IncomingBytes":          c.queryMetricName("IncomingBytes"),
				"Duration":               c.queryMetricName("Duration"),
				"BytesReadIntoMemcached": c.queryMetricName("BytesReadIntoMemcached"),
				"NetworkBytesIn":         c.queryMetricName("NetworkBytesIn"),
				"GetHits":                c.queryMetricName("GetHits"),
				"ApproximateNumberOfMessagesDelayed": c.queryMetricName("ApproximateNumberOfMessagesDelayed"),
				"BytesWrittenOutFromMemcached":       c.queryMetricName("BytesWrittenOutFromMemcached"),
				"Errors":                             c.queryMetricName("Errors"),
				"DeleteMisses":                       c.queryMetricName("DeleteMisses"),
				"CPUReservation":                     c.queryMetricName("CPUReservation"),
				"CasBadval":                          c.queryMetricName("CasBadval"),
				"MatchedEvents":                      c.queryMetricName("MatchedEvents"),
			},
		}),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			_, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			_, ok = p.Context.Value(queryContextKey).(*QueryContext)
			if !ok {
				return nil, errDecodeQueryContext
			}

			var (
				instanceId    string
				namespace     string
				dimensionName string
			)

			switch t := p.Source.(type) {
			case *opsee_aws_ec2.Instance:
				instanceId = aws.StringValue(t.InstanceId)
				namespace = "AWS/EC2"
				dimensionName = "InstanceId"
			case *opsee_aws_rds.DBInstance:
				instanceId = aws.StringValue(t.DBInstanceIdentifier)
				namespace = "AWS/RDS"
				dimensionName = "DBInstanceIdentifier"
			default:
				return nil, errUnknownInstanceMetricType
			}

			var (
				interval  = 3600
				period    = 60
				endTime   = time.Now().UTC().Add(time.Duration(-1) * time.Minute) // 1 minute lag.  otherwise we won't get stats
				startTime = endTime.Add(time.Duration(-1*interval) * time.Second)
				startTs   = &opsee_types.Timestamp{}
				endTs     = &opsee_types.Timestamp{}
			)

			startTs.Scan(startTime)
			endTs.Scan(endTime)

			return &opsee_aws_cloudwatch.GetMetricStatisticsInput{
				StartTime:  startTs,
				EndTime:    endTs,
				Period:     aws.Int64(int64(period)),
				Namespace:  aws.String(namespace),
				Statistics: []string{"Average"},
				Dimensions: []*opsee_aws_cloudwatch.Dimension{
					{
						Name:  aws.String(dimensionName),
						Value: aws.String(instanceId),
					},
				},
			}, nil
		},
	}
}

func (c *Composter) queryMetricName(metricName string) *graphql.Field {
	return &graphql.Field{
		Type: schema.GraphQLCloudWatchResponseType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			queryContext, ok := p.Context.Value(queryContextKey).(*QueryContext)
			if !ok {
				return nil, errDecodeQueryContext
			}

			input, ok := p.Source.(*opsee_aws_cloudwatch.GetMetricStatisticsInput)
			if !ok {
				return nil, errDecodeMetricStatisticsInput
			}

			input.MetricName = aws.String(metricName)

			return c.resolver.GetMetricStatistics(p.Context, user, queryContext.Region, input)
		},
	}
}

func (c *Composter) mutation() *graphql.Object {
	mutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"checks":       c.upsertChecks(),
			"deleteChecks": c.deleteChecks(),
			"testCheck":    c.testCheck(),
		},
	})

	return mutation
}

func (c *Composter) upsertChecks() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(CheckType),
		Args: graphql.FieldConfigArgument{
			"checks": &graphql.ArgumentConfig{
				Description: "A list of checks to create",
				Type:        graphql.NewList(CheckInputType),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			checksInput, ok := p.Args["checks"].([]interface{})
			if !ok {
				return nil, errDecodeCheckInput
			}
			return c.resolver.UpsertChecks(p.Context, user, checksInput)
		},
	}
}

func (c *Composter) deleteChecks() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(graphql.String),
		Args: graphql.FieldConfigArgument{
			"ids": &graphql.ArgumentConfig{
				Description: "A list of check ids to delete",
				Type:        graphql.NewList(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			checksInput, ok := p.Args["ids"].([]interface{})
			if !ok {
				return nil, errDecodeCheckInput
			}

			return c.resolver.DeleteChecks(p.Context, user, checksInput)
		},
	}
}

func (c *Composter) testCheck() *graphql.Field {
	return &graphql.Field{
		Type: opsee.GraphQLTestCheckResponseType,
		Args: graphql.FieldConfigArgument{
			"check": &graphql.ArgumentConfig{
				Description: "A test check",
				Type:        CheckInputType,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			checkInput, ok := p.Args["check"].(map[string]interface{})
			if !ok {
				return nil, errDecodeCheckInput
			}

			return c.resolver.TestCheck(p.Context, user, checkInput)
		},
	}
}

func addFields(obj *graphql.Object, fields graphql.FieldDefinitionMap) {
	for fname, f := range fields {
		obj.AddFieldConfig(fname, &graphql.Field{
			Name:        f.Name,
			Description: f.Description,
			Type:        f.Type,
			Resolve:     f.Resolve,
		})
	}
}
