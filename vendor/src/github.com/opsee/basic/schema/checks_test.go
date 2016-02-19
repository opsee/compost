package schema

import (
	"github.com/graphql-go/graphql"
	"math/rand"
	"testing"
	"time"
)

func TestCheckschema(t *testing.T) {
	popr := rand.New(rand.NewSource(time.Now().UnixNano()))
	check := NewPopulatedCheck(popr, false)

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"check": &graphql.Field{
					Type: GraphQLCheckType,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return check, nil
					},
				},
			},
		}),
	})

	if err != nil {
		t.Fatal(err)
	}

	queryResponse := graphql.Do(graphql.Params{Schema: schema, RequestString: `query yeOldeQuery {
		check {
			id
			interval
			target {
				name
				type
				id
				address
			}
			last_run
			spec {
				... on schemaHttpCheck {
					name
					path
					protocol
					port
					verb
					headers {
						name
						values
					}
					body
				}
				... on schemaCloudWatchCheck {
					target {
						name
						type
						id
						address
					}
					metric_name
					function
					function_params
				}
			}
			name
			assertions {
				key
				value
				relationship
				operand
			}
			results {
				check_id
				customer_id
				timestamp
				passing
				responses {
					target {
						name
						type
						id
						address
					}
					response
					error
					passing
					reply {
						... on schemaHttpResponse {
							code
							body
							headers {
								name
								values
							}
							metrics {
								name
								value
								tags
								timestamp
							}
							host
						}
					}
				}
				target {
					name
					type
					id
					address
				}
				check_name
				version
			}
		}
	}`})

	if queryResponse.HasErrors() {
		t.Fatalf("graphql query errors: %#v\n", queryResponse.Errors)
	}
}
