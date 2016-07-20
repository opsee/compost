package composter

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/graphql-go/graphql"
	"github.com/opsee/basic/schema"
	opsee_aws_autoscaling "github.com/opsee/basic/schema/aws/autoscaling"
	opsee_aws_cloudwatch "github.com/opsee/basic/schema/aws/cloudwatch"
	opsee_aws_ec2 "github.com/opsee/basic/schema/aws/ec2"
	opsee_aws_ecs "github.com/opsee/basic/schema/aws/ecs"
	opsee_aws_elb "github.com/opsee/basic/schema/aws/elb"
	opsee_aws_rds "github.com/opsee/basic/schema/aws/rds"
	opsee "github.com/opsee/basic/service"
	log "github.com/opsee/logrus"
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
	opsee_scalars "github.com/opsee/protobuf/plugin/graphql/scalars"
)

var (
	errDecodeUser                  = errors.New("error decoding user")
	errDecodeQueryContext          = errors.New("error decoding query context")
	errMissingRegion               = errors.New("missing region id")
	errMissingVpc                  = errors.New("missing vpc id")
	errMissingService              = errors.New("missing service name")
	errMissingInstanceType         = errors.New("missing instance type - must be one of (ec2, rds)")
	errMissingGroupType            = errors.New("missing group type - must be one of (security, elb, autoscaling)")
	errDecodeInstances             = errors.New("error decoding instances")
	errDecodeInstanceIds           = errors.New("error decoding instance ids")
	errDecodeUserPermissions       = errors.New("error decoding permissions")
	errUnknownInstanceMetricType   = errors.New("no metrics for that instance type")
	errDecodeMetricStatisticsInput = errors.New("error decoding metric statistics input")
	errDecodeCheckInput            = errors.New("error decoding checks input")
	errDecodeTeamInput             = errors.New("error decoding team input")
	errDecodeUserInput             = errors.New("error decoding user input")
	errDecodeNotificationsInput    = errors.New("error decoding notifications input")
	errUnknownAction               = errors.New("unknown action")

	UserStatusEnumType       *graphql.Enum
	TeamSubscriptionEnumType *graphql.Enum
	AggregationEnumType      *graphql.Enum

	InstanceType   *graphql.Object
	DbInstanceType *graphql.Object
	EcsServiceType *graphql.Object
	CheckType      *graphql.Object

	CheckInputType        *graphql.InputObject
	TeamInputType         *graphql.InputObject
	UserInputType         *graphql.InputObject
	UserFlagsInputType    *graphql.InputObject
	NotificationInputType *graphql.InputObject
	AggregationInputType  *graphql.InputObject
)

type instanceAction int

const (
	instanceReboot instanceAction = iota
	instanceStart
	instanceStop
)

type PermissionOp struct {
	op   string
	perm string
}

func (p PermissionOp) Op(user *schema.User, has error) error {
	switch p.op {
	case "and":
		if has != nil {
			return has
		}
		return user.CheckPermission(p.perm)
	case "or":
		has = user.CheckPermission(p.perm)
	default:
		return fmt.Errorf("invalid op: %s", p.op)
	}
	return has
}

func UserPermittedFromContext(ctx context.Context, perm string, extra ...PermissionOp) (*schema.User, error) {
	log.Debugf("checking permission %v for context %v", ctx, perm)
	user, ok := ctx.Value(userKey).(*schema.User)
	if !ok || user == nil {
		return nil, errDecodeUser
	}

	has := user.CheckPermission(perm)
	for _, op := range extra {
		has = op.Op(user, has)
		if has != nil {
			return user, has
		}
	}

	return user, has
}

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
	if UserStatusEnumType == nil {
		UserStatusEnumType = graphql.NewEnum(graphql.EnumConfig{
			Name: "UserStatus",
			Values: graphql.EnumValueConfigMap{
				"invited": &graphql.EnumValueConfig{
					Value: "invited",
				},
				"active": &graphql.EnumValueConfig{
					Value: "active",
				},
				"inactive": &graphql.EnumValueConfig{
					Value: "inactive",
				},
			},
		})
	}

	if TeamSubscriptionEnumType == nil {
		TeamSubscriptionEnumType = graphql.NewEnum(graphql.EnumConfig{
			Name: "TeamSubscription",
			Values: graphql.EnumValueConfigMap{
				"free": &graphql.EnumValueConfig{
					Value: "free",
				},
				"developer_monthly": &graphql.EnumValueConfig{
					Value: "developer_monthly",
				},
				"team_monthly": &graphql.EnumValueConfig{
					Value: "team_monthly",
				},
			},
		})
	}

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

	if EcsServiceType == nil {
		EcsServiceType = graphql.NewObject(graphql.ObjectConfig{
			Name: opsee_aws_ecs.GraphQLServiceType.Name(),
			Fields: graphql.Fields{
				"metrics": metrics,
			},
		})
		addFields(EcsServiceType, opsee_aws_ecs.GraphQLServiceType.Fields())
	}

	if AggregationEnumType == nil {
		AggregationEnumType = graphql.NewEnum(graphql.EnumConfig{
			Name: "AggregationEnum",
			Values: graphql.EnumValueConfigMap{
				"avg": &graphql.EnumValueConfig{
					Value: "avg",
				},
				"sum": &graphql.EnumValueConfig{
					Value: "sum",
				},
				"min": &graphql.EnumValueConfig{
					Value: "min",
				},
				"max": &graphql.EnumValueConfig{
					Value: "max",
				},
			},
		})
	}

	if AggregationInputType == nil {
		AggregationInputType = graphql.NewInputObject(graphql.InputObjectConfig{
			Name:        "Aggregation",
			Description: "Metrics Aggregation",
			Fields: graphql.InputObjectConfigFieldMap{
				"unit": &graphql.InputObjectFieldConfig{
					Type:        graphql.String,
					Description: "Unit of time (milliseconds, seconds, minutes...)",
				},
				"period": &graphql.InputObjectFieldConfig{
					Type:        graphql.Int,
					Description: "Period over which to aggregate",
				},
				"type": &graphql.InputObjectFieldConfig{
					Type:        AggregationEnumType,
					Description: "sum, avg, min, max etc",
				},
			},
		})
	}

	checkStateTransitions := c.queryCheckStateTransitions()
	checkMetrics := c.queryCheckMetrics()
	if CheckType == nil {
		CheckType = graphql.NewObject(graphql.ObjectConfig{
			Name: schema.GraphQLCheckType.Name(),
			Fields: graphql.Fields{
				"notifications": &graphql.Field{
					Type: graphql.NewList(schema.GraphQLNotificationType),
				},
				"metrics":           checkMetrics,
				"state_transitions": checkStateTransitions,
			},
		})
		addFields(CheckType, schema.GraphQLCheckType.Fields())
	}

	if TeamInputType == nil {
		TeamInputType = graphql.NewInputObject(graphql.InputObjectConfig{
			Name:        "Team",
			Description: "An Opsee Team",
			Fields: graphql.InputObjectConfigFieldMap{
				"name": &graphql.InputObjectFieldConfig{
					Type:        graphql.String,
					Description: "The team name",
				},
				"plan": &graphql.InputObjectFieldConfig{
					Type:        TeamSubscriptionEnumType,
					Description: "The plan",
				},
				"stripeToken": &graphql.InputObjectFieldConfig{
					Type:        graphql.String,
					Description: "The credit card token",
				},
			},
		})
	}

	if UserFlagsInputType == nil {
		UserFlagsInputType = graphql.NewInputObject(graphql.InputObjectConfig{
			Name:        "UserFlags",
			Description: "An Opsee Team",
			Fields: graphql.InputObjectConfigFieldMap{
				"admin": &graphql.InputObjectFieldConfig{
					Type:        graphql.Boolean,
					Description: "Administrator access",
				},
				"edit": &graphql.InputObjectFieldConfig{
					Type:        graphql.Boolean,
					Description: "Edit access",
				},
				"billing": &graphql.InputObjectFieldConfig{
					Type:        graphql.Boolean,
					Description: "Billing access",
				},
			},
		})
	}

	if UserInputType == nil {
		UserInputType = graphql.NewInputObject(graphql.InputObjectConfig{
			Name:        "User",
			Description: "An Opsee User",
			Fields: graphql.InputObjectConfigFieldMap{
				"id": &graphql.InputObjectFieldConfig{
					Type:        graphql.NewNonNull(graphql.Int),
					Description: "The user id",
				},
				"email": &graphql.InputObjectFieldConfig{
					Type:        graphql.String,
					Description: "The user's email",
				},
				"name": &graphql.InputObjectFieldConfig{
					Type:        graphql.String,
					Description: "The user's name",
				},
				"status": &graphql.InputObjectFieldConfig{
					Type:        graphql.NewNonNull(UserStatusEnumType),
					Description: "The user's status",
				},
				"perms": &graphql.InputObjectFieldConfig{
					Description: "A list of user permissions",
					Type:        UserFlagsInputType,
				},
			},
		})
	}

	if NotificationInputType == nil {
		NotificationInputType = graphql.NewInputObject(graphql.InputObjectConfig{
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
		})
	}

	if CheckInputType == nil {
		CheckInputType = graphql.NewInputObject(graphql.InputObjectConfig{
			Name:        "Check",
			Description: "An Opsee Check",
			Fields: graphql.InputObjectConfigFieldMap{
				"id": &graphql.InputObjectFieldConfig{
					Type:        graphql.String,
					Description: "The check id",
				},
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
					Type:        graphql.NewNonNull(graphql.NewList(NotificationInputType)),
					Description: "Check notifications",
				},
				"min_failing_count": &graphql.InputObjectFieldConfig{
					Type:        graphql.Int,
					Description: "How many nodes must fail in order for a check to fail",
				},
				"min_failing_time": &graphql.InputObjectFieldConfig{
					Type:        graphql.Int,
					Description: "How long (in seconds) must a check fail in order to be considered failing",
				},
			},
		})
	}
}

func (c *Composter) query() *graphql.Object {
	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"checks":        c.queryChecks(),
			"region":        c.queryRegion(),
			"hasRole":       c.queryHasRole(),
			"role":          c.queryRole(),
			"team":          c.queryTeam(),
			"notifications": c.queryNotifications(),
		},
	})

	return query
}

func (c *Composter) queryCheckStateTransitions() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(schema.GraphQLCheckStateTransitionType),
		Args: graphql.FieldConfigArgument{
			"start_time": &graphql.ArgumentConfig{
				Description: "unix timestamp start time",
				Type:        opsee_scalars.Timestamp,
			},
			"end_time": &graphql.ArgumentConfig{
				Description: "unix timestmap end time",
				Type:        opsee_scalars.Timestamp,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			check, ok := p.Source.(*schema.Check)
			if !ok {
				return nil, fmt.Errorf("missing check id")
			}
			checkId := check.Id

			var (
				startTime *opsee_types.Timestamp
				endTime   *opsee_types.Timestamp
			)

			startTime = &opsee_types.Timestamp{}
			endTime = &opsee_types.Timestamp{}

			ts0, _ := p.Args["start_time"].(int)
			ts1, _ := p.Args["end_time"].(int)

			_ = startTime.Scan(ts0)
			_ = endTime.Scan(ts1)

			return c.resolver.GetCheckStateTransitions(p.Context, user, checkId, startTime, endTime)
		},
	}
}

func (c *Composter) queryCheckMetrics() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(schema.GraphQLMetricType),
		Args: graphql.FieldConfigArgument{
			"metric_name": &graphql.ArgumentConfig{
				Description: "name of the metric",
				Type:        graphql.String,
			},
			"start_time": &graphql.ArgumentConfig{
				Description: "unix timestamp start time",
				Type:        opsee_scalars.Timestamp,
			},
			"end_time": &graphql.ArgumentConfig{
				Description: "unix timestmap end time",
				Type:        opsee_scalars.Timestamp,
			},
			"aggregation": &graphql.ArgumentConfig{
				Description: "aggregator",
				Type:        AggregationInputType,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			check, ok := p.Source.(*schema.Check)
			if !ok {
				return nil, fmt.Errorf("missing check id")
			}
			checkId := check.Id

			var (
				metricName string
				startTime  *opsee_types.Timestamp
				endTime    *opsee_types.Timestamp
			)
			startTime = &opsee_types.Timestamp{}
			endTime = &opsee_types.Timestamp{}

			metricName, _ = p.Args["metric_name"].(string)
			ts0, _ := p.Args["start_time"].(int)
			ts1, _ := p.Args["end_time"].(int)

			_ = startTime.Scan(ts0)
			_ = endTime.Scan(ts1)

			var aggregator *opsee.Aggregation
			if ag, ok := p.Args["aggregation"].(map[string]interface{}); ok {
				u, _ := ag["unit"].(string)
				p, _ := ag["period"].(int)
				a, _ := ag["type"].(string)

				aggregator = &opsee.Aggregation{
					Unit:   u,
					Period: int64(p),
					Type:   a,
				}
			}

			return c.resolver.GetCheckMetrics(p.Context, user, checkId, metricName, startTime, endTime, aggregator)
		},
	}
}

func (c *Composter) adminQuery() *graphql.Object {
	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"checks":        c.queryChecks(),
			"region":        c.queryRegion(),
			"role":          c.queryRole(),
			"team":          c.queryTeam(),
			"notifications": c.queryNotifications(),
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
					requestor, err := UserPermittedFromContext(p.Context, opsee_types.OpseeAdmin)
					if err != nil {
						return nil, err
					}

					var (
						page    int
						perPage int
					)

					page, _ = p.Args["page"].(int)
					perPage, _ = p.Args["per_page"].(int)

					return c.resolver.ListCustomers(p.Context, &opsee.ListUsersRequest{
						Requestor: requestor,
						Page:      int32(page),
						PerPage:   int32(perPage),
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
					requestor, err := UserPermittedFromContext(p.Context, opsee_types.OpseeAdmin)
					if err != nil {
						return nil, err
					}
					var (
						customerId string
						email      string
						id         int
					)

					customerId, _ = p.Args["customer_id"].(string)
					email, _ = p.Args["email"].(string)
					id, _ = p.Args["id"].(int)

					return c.resolver.GetUser(p.Context, &opsee.GetUserRequest{
						Requestor:  requestor,
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

func (c *Composter) queryHasRole() *graphql.Field {
	return &graphql.Field{
		Type: graphql.Boolean,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			return c.resolver.HasRole(p.Context, user)
		},
	}
}

func (c *Composter) queryRole() *graphql.Field {
	return &graphql.Field{
		Type: schema.GraphQLRoleStackType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			return c.resolver.GetRoleStack(p.Context, user)
		},
	}
}

func (c *Composter) queryTeam() *graphql.Field {
	return &graphql.Field{
		Type: schema.GraphQLTeamType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			return c.resolver.GetTeam(p.Context, user)
		},
	}
}

func (c *Composter) queryNotifications() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(schema.GraphQLNotificationType),
		Args: graphql.FieldConfigArgument{
			"default": &graphql.ArgumentConfig{
				Description: "Fetch default notifications",
				Type:        graphql.Boolean,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			defaultOnly, _ := p.Args["default"].(bool)

			return c.resolver.GetNotifications(p.Context, user, defaultOnly)
		},
	}
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
				"vpc":             c.queryVpc(),
				"task_definition": c.queryTaskDefinition(),
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

func (c *Composter) queryTaskDefinition() *graphql.Field {
	return &graphql.Field{
		Type: opsee_aws_ecs.GraphQLTaskDefinitionType,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "The Task Definition id",
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

			id, _ := p.Args["id"].(string)
			if id == "" {
				return nil, errMissingService
			}

			return c.resolver.GetTaskDefinition(p.Context, user, queryContext.Region, id)
		},
	}
}

func (c *Composter) queryVpc() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewObject(graphql.ObjectConfig{
			Name:        "VPC",
			Description: "An AWS VPC",
			Fields: graphql.Fields{
				"groups":    c.queryGroups(),
				"instances": c.queryInstances(),
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
				EcsServiceType,
				opsee_aws_elb.GraphQLLoadBalancerDescriptionType,
				opsee_aws_autoscaling.GraphQLGroupType,
			},
			ResolveType: func(value interface{}, info graphql.ResolveInfo) *graphql.Object {
				switch value.(type) {
				case *opsee_aws_ec2.SecurityGroup:
					return opsee_aws_ec2.GraphQLSecurityGroupType
				case *opsee_aws_ecs.Service:
					return EcsServiceType
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
				Type:        graphql.String,
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

			groups, err := c.resolver.GetGroups(p.Context, user, queryContext.Region, queryContext.VpcId, groupType, groupId)
			if err != nil {
				log.WithError(err).Error("error querying groups")
				return nil, err
			}

			return groups, nil
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
			requestor, err := UserPermittedFromContext(p.Context, "admin")
			if err != nil {
				return nil, err
			}
			user := requestor
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
				namespace  string
				dimensions []*opsee_aws_cloudwatch.Dimension
			)

			switch t := p.Source.(type) {
			case *opsee_aws_ec2.Instance:
				namespace = "AWS/EC2"
				dimensions = []*opsee_aws_cloudwatch.Dimension{
					{
						Name:  aws.String("InstanceId"),
						Value: t.InstanceId,
					},
				}
			case *opsee_aws_rds.DBInstance:
				namespace = "AWS/RDS"
				dimensions = []*opsee_aws_cloudwatch.Dimension{
					{
						Name:  aws.String("DBInstanceIdentifier"),
						Value: t.DBInstanceIdentifier,
					},
				}
			case *opsee_aws_ecs.Service:
				clustername, err := clusterNameFromArn(t.ClusterArn)
				if err != nil {
					return nil, err
				}

				namespace = "AWS/ECS"
				dimensions = []*opsee_aws_cloudwatch.Dimension{
					{
						Name:  aws.String("ClusterName"),
						Value: clustername,
					},
					{
						Name:  aws.String("ServiceName"),
						Value: t.ServiceName,
					},
				}
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
				Dimensions: dimensions,
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
			"checks":                    c.upsertChecks(),
			"deleteChecks":              c.deleteChecks(),
			"testCheck":                 c.testCheck(),
			"makeLaunchRoleUrlTemplate": c.makeLaunchRoleUrlTemplate(),
			"makeLaunchRoleUrl":         c.makeLaunchRoleUrl(),
			"region":                    c.mutateRegion(),
			"team":                      c.mutateTeam(),
			"user":                      c.mutateUser(),
			"notifications":             c.mutateNotifications(),
		},
	})

	return mutation
}

func (c *Composter) mutateNotifications() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(schema.GraphQLNotificationType),
		Args: graphql.FieldConfigArgument{
			"default": &graphql.ArgumentConfig{
				Description: "Default notifications to set",
				Type:        graphql.NewList(NotificationInputType),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			requestor, err := UserPermittedFromContext(p.Context, "admin", PermissionOp{"or", "edit"})
			if err != nil {
				return nil, err
			}

			notificationsInput, ok := p.Args["default"].([]interface{})
			if !ok {
				return nil, errDecodeNotificationsInput
			}

			return c.resolver.PutDefaultNotifications(p.Context, requestor, notificationsInput)
		},
	}
}

func (c *Composter) mutateTeam() *graphql.Field {
	return &graphql.Field{
		Type: schema.GraphQLTeamType,
		Args: graphql.FieldConfigArgument{
			"team": &graphql.ArgumentConfig{
				Description: "The Team to update",
				Type:        TeamInputType,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			requestor, err := UserPermittedFromContext(p.Context, "admin")
			if err != nil {
				return nil, err
			}

			teamInput, ok := p.Args["team"].(map[string]interface{})
			if !ok {
				return nil, errDecodeTeamInput
			}

			return c.resolver.PutTeam(p.Context, requestor, teamInput)
		},
	}
}

func (c *Composter) mutateUser() *graphql.Field {
	return &graphql.Field{
		Type: schema.GraphQLUserType,
		Args: graphql.FieldConfigArgument{
			"user": &graphql.ArgumentConfig{
				Description: "The User to update",
				Type:        UserInputType,
			},
			"password": &graphql.ArgumentConfig{
				Description: "The user's new password",
				Type:        graphql.String,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			// TODO(dan) only admins mutate users rn
			requestor, err := UserPermittedFromContext(p.Context, "admin")
			if err != nil {
				return nil, err
			}

			userInput, ok := p.Args["user"].(map[string]interface{})
			if !ok {
				return nil, errDecodeUserInput
			}
			password, _ := p.Args["password"].(string)

			log.Debugf("user %v", p.Args["user"])
			newUser := &schema.User{}

			tb, err := json.Marshal(userInput)
			if err != nil {
				log.WithError(err).Error("marshal user input")
				return nil, errDecodeUserInput
			}

			err = json.Unmarshal(tb, newUser)
			if err != nil {
				log.WithError(err).Error("unmarshal user input")
				return nil, errDecodeUserInput
			}

			log.Debugf("unmarshalled user %v", newUser)

			// invites
			if newUser.Id == 0 && newUser.Email != "" && newUser.Perms != nil {
				req := &opsee.InviteUserRequest{
					Requestor: requestor,
					Email:     newUser.Email,
					Perms:     newUser.Perms,
				}

				return c.resolver.InviteUser(p.Context, req)
			}

			req := &opsee.UpdateUserRequest{
				Requestor: requestor,
				User: &schema.User{
					Id:         newUser.Id,
					CustomerId: requestor.CustomerId,
				},
				Email:    newUser.Email,
				Name:     newUser.Name,
				Status:   newUser.Status,
				Perms:    newUser.Perms,
				Password: password,
			}

			return c.resolver.PutUser(p.Context, req)
		},
	}
}

func (c *Composter) mutateRegion() *graphql.Field {

	return &graphql.Field{
		Type: graphql.NewObject(graphql.ObjectConfig{
			Name:        "RegionMutation",
			Description: "The AWS Region",
			Fields: graphql.Fields{
				"rebootInstances": c.instanceAction(instanceReboot),
				"startInstances":  c.instanceAction(instanceStart),
				"stopInstances":   c.instanceAction(instanceStop),
				"scan":            c.scanRegion(),
				"launchStack":     c.launchStack(),
			},
		}),
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "The region id",
				Type:        graphql.NewNonNull(graphql.String),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			_, err := UserPermittedFromContext(p.Context, "admin", PermissionOp{"or", "edit"})
			if err != nil {
				return nil, err
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

func (c *Composter) launchStack() *graphql.Field {
	return &graphql.Field{
		Type: graphql.Boolean,
		Args: graphql.FieldConfigArgument{
			"vpc_id": &graphql.ArgumentConfig{
				Description: "The VPC id",
				Type:        graphql.NewNonNull(graphql.String),
			},
			"subnet_id": &graphql.ArgumentConfig{
				Description: "The Subnet id",
				Type:        graphql.NewNonNull(graphql.String),
			},
			"subnet_routing": &graphql.ArgumentConfig{
				Description: "The Subnet routing",
				Type:        graphql.NewNonNull(graphql.String),
			},
			"instance_size": &graphql.ArgumentConfig{
				Description: "The AWS instance size",
				Type:        graphql.String,
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

			if queryContext.Region == "" {
				return nil, errMissingRegion
			}

			vpcId, _ := p.Args["vpc_id"].(string)
			subnetId, _ := p.Args["subnet_id"].(string)
			subnetRouting, _ := p.Args["subnet_routing"].(string)
			instanceSize, _ := p.Args["instance_size"].(string)
			if instanceSize == "" {
				instanceSize = "t2.micro"
			}

			return c.resolver.LaunchBastionStack(p.Context, user, queryContext.Region, vpcId, subnetId, subnetRouting, instanceSize)
		},
	}
}

func (c *Composter) scanRegion() *graphql.Field {
	return &graphql.Field{
		Type: schema.GraphQLRegionType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			queryContext, ok := p.Context.Value(queryContextKey).(*QueryContext)
			if !ok {
				return nil, errDecodeQueryContext
			}

			if queryContext.Region == "" {
				return nil, errMissingRegion
			}

			return c.resolver.ScanRegion(p.Context, user, queryContext.Region)
		},
	}
}

func (c *Composter) makeLaunchRoleUrlTemplate() *graphql.Field {
	return &graphql.Field{
		Type: JsonScalar,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			templ, err := c.resolver.LaunchRoleUrlTemplate(p.Context, user)
			if err != nil {
				return nil, err
			}

			// returning as a json.RawMessage so as to not escape the ampersand
			return json.RawMessage(templ), nil
		},
	}
}

func (c *Composter) makeLaunchRoleUrl() *graphql.Field {
	return &graphql.Field{
		Type: JsonScalar,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			templ, err := c.resolver.LaunchRoleUrl(p.Context, user)
			if err != nil {
				return nil, err
			}

			// returning as a json.RawMessage so as to not escape the ampersand
			return json.RawMessage(templ), nil
		},
	}
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
			// must have admin or edit to upsert checks
			_, err := UserPermittedFromContext(p.Context, "admin", PermissionOp{"or", "edit"})
			if err != nil {
				return nil, err
			}

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
			// must have admin or edit to delete checks
			requestor, err := UserPermittedFromContext(p.Context, "admin", PermissionOp{"or", "edit"})
			if err != nil {
				return nil, err
			}

			checksInput, ok := p.Args["ids"].([]interface{})
			if !ok {
				return nil, errDecodeCheckInput
			}

			return c.resolver.DeleteChecks(p.Context, requestor, checksInput)
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
			// TODO(dan) not sure about this one
			requestor, err := UserPermittedFromContext(p.Context, "admin", PermissionOp{"or", "edit"})
			if err != nil {
				return nil, err
			}

			checkInput, ok := p.Args["check"].(map[string]interface{})
			if !ok {
				return nil, errDecodeCheckInput
			}

			return c.resolver.TestCheck(p.Context, requestor, checkInput)
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

func clusterNameFromArn(arn *string) (*string, error) {
	arnParts := strings.Split(aws.StringValue(arn), "/")
	if len(arnParts) < 2 {
		return nil, fmt.Errorf("invalid cluster ARN")
	}

	return aws.String(arnParts[len(arnParts)-1]), nil
}
