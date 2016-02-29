package composter

import (
	"errors"
	"fmt"
	"github.com/graphql-go/graphql"
	"github.com/opsee/basic/schema"
	opsee "github.com/opsee/basic/service"
	// log "github.com/sirupsen/logrus"
)

var (
	errDecodeUser = errors.New("error decoding user")
)

func (c *Composter) mustSchema() {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: c.query(),
	})

	if err != nil {
		panic(fmt.Sprint("error generating graphql schema: ", err))
	}

	adminSchema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: c.adminQuery(),
	})

	if err != nil {
		panic(fmt.Sprint("error generating graphql schema: ", err))
	}

	c.Schema = schema
	c.AdminSchema = adminSchema
}

func (c *Composter) query() *graphql.Object {
	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"checks": c.queryChecks(),
		},
	})

	return query
}

func (c *Composter) adminQuery() *graphql.Object {
	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"listChecks": c.queryChecks(),
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
		Type: graphql.NewList(schema.GraphQLCheckType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			user, ok := p.Context.Value(userKey).(*schema.User)
			if !ok {
				return nil, errDecodeUser
			}

			return c.resolver.ListChecks(p.Context, user)
		},
	}
}
