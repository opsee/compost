package composter

import (
	"encoding/json"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

func coerceJson(value interface{}) interface{} {
	switch t := value.(type) {
	case json.RawMessage:
		return t
	case []byte:
		return json.RawMessage(t)
	}
	return nil
}

var JsonScalar *graphql.Scalar = graphql.NewScalar(graphql.ScalarConfig{
	Name:       "JsonRawMessage",
	Serialize:  coerceJson,
	ParseValue: coerceJson,
	ParseLiteral: func(valueAST ast.Value) interface{} {
		switch valueAST := valueAST.(type) {
		case *ast.StringValue:
			return valueAST.Value
		}
		return nil
	},
})
