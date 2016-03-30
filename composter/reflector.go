package composter

import (
	"fmt"
	"github.com/graphql-go/graphql"
	"reflect"
	"strings"
	"time"
)

func GraphQLStructObject(thing interface{}, name, description string) *graphql.Object {
	fields := make(graphql.Fields)
	reflectType := reflect.TypeOf(thing)

	for i := 0; i < reflectType.NumField(); i++ {
		structField := reflectType.Field(i)

		// bail if there is a package path, since that's an unexported field
		if structField.PkgPath != "" {
			continue
		}

		// name the graphQL field after the struct field name
		fieldName := structField.Name

		// if we find an explicit json field name, use that instead
		jsonTag := strings.Split(structField.Tag.Get("json"), ",")
		if len(jsonTag) > 0 && jsonTag[0] != "" {
			fieldName = jsonTag[0]
		}

		// pull the description from the "api" struct tag
		description := structField.Tag.Get("api")

		fields[fieldName] = &graphql.Field{
			Type:        graphQLFieldOutput(structField.Type, fieldName, description),
			Description: description,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				v := reflect.ValueOf(p.Source)
				if v.Kind() == reflect.Ptr {
					v = reflect.Indirect(v)
				}

				v = v.FieldByName(structField.Name)
				if v.Kind() == reflect.Ptr {
					v = reflect.Indirect(v)
				}

				return v.Interface(), nil
			},
		}
	}

	return graphql.NewObject(graphql.ObjectConfig{
		Name:        name,
		Description: description,
		Fields:      fields,
	})
}

func graphQLFieldOutput(t reflect.Type, fieldName, description string) graphql.Output {
	var gqlType graphql.Output
	k := t.Kind()

	switch k {
	case reflect.String:
		gqlType = graphql.String

	case reflect.Bool:
		gqlType = graphql.Boolean

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		gqlType = graphql.Int

	case reflect.Float32, reflect.Float64:
		gqlType = graphql.Float

	case reflect.Struct:
		// special case time.Time
		if t == reflect.TypeOf(time.Time{}) {
			gqlType = graphql.String // should we make this a custom scalar type? i don't really care rn, .String() works
			break
		}

		gqlType = GraphQLStructObject(reflect.Zero(t).Interface(), fieldName, description)

	case reflect.Ptr:
		gqlType = graphQLFieldOutput(t.Elem(), fieldName, description)

	case reflect.Slice:
		gqlType = graphql.NewList(graphQLFieldOutput(t.Elem(), fieldName, description))

	default:
		panic(fmt.Sprintf("name: type %s: %s not mapped to graphql type, define in composter/reflector.go", fieldName, k.String()))
	}

	return gqlType
}
