package composter

import (
	"errors"
	"fmt"
	"github.com/graphql-go/graphql"
	"github.com/opsee/basic/schema"
	// log "github.com/sirupsen/logrus"
)

var (
	errDecodeUser = errors.New("error decoding user")
)

func (c *Composter) mustSchema() {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: c.genQuery(),
	})

	if err != nil {
		panic(fmt.Sprint("error generating graphql schema: ", err))
	}

	c.Schema = schema
}

func (c *Composter) genQuery() *graphql.Object {
	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"checks": &graphql.Field{
				Type: graphql.NewList(schema.GraphQLCheckType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					user, ok := p.Context.Value(userKey).(*schema.User)
					if !ok {
						return nil, errDecodeUser
					}

					return c.resolver.ResolveChecks(user)
				},
			},
			// "groups",
			// "instance",
			// "instances",
		},
	})

	return query
}

// Args: graphql.FieldConfigArgument{
// 	"id": &graphql.ArgumentConfig{
// 		Description: "The id of the group.",
// 		Type:        graphql.NewNonNull(graphql.String),
// 	},
// 	"type": &graphql.ArgumentConfig{
// 		Description: "The type of the group.",
// 		Type:        graphql.NewNonNull(graphql.String),
// 	},
// },
