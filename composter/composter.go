// Package composter is a graphql http server, and is responsible for defining
// the overall graphql schema.
package composter

import (
	"errors"
	"github.com/graphql-go/graphql"
	"github.com/opsee/basic/tp"
	"github.com/opsee/compost/resolver"
	"golang.org/x/net/context"
)

const (
	userKey = iota
	requestKey
	resolverKey
)

var (
	errDecodeRequest = errors.New("error decoding request from context")
	errNoQuery       = errors.New("query not provided")
)

type Composter struct {
	Schema      graphql.Schema
	AdminSchema graphql.Schema
	resolver    resolver.Client
	router      *tp.Router
}

func New(resolver resolver.Client) *Composter {
	composter := &Composter{
		resolver: resolver,
	}

	composter.mustSchema()
	composter.initHTTP()
	return composter
}

func (c *Composter) Compost(ctx context.Context, schema graphql.Schema) (*graphql.Result, error) {
	request, ok := ctx.Value(requestKey).(*GraphQLRequest)
	if !ok {
		return nil, errDecodeRequest
	}

	response := graphql.Do(graphql.Params{
		Schema:         schema,
		RequestString:  request.Query,
		VariableValues: request.Variables,
		Context:        ctx,
	})

	return response, nil
}

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

func (req *GraphQLRequest) Validate() error {
	if req.Query == "" {
		return errNoQuery
	}

	return nil
}
