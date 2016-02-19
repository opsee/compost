package composter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/graphql-go/graphql"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type testNested struct {
	Name    string  `json:"name" api:"The nested name"`
	NamePtr *string `json:"name_ptr" api:"Pointer to the nested name"`
	IntData int     `json:"int_data" api:"The nested data"`
}

type testGroup struct {
	Name          string `json:"name" api:"The group identifier"`
	CustomerID    string `json:"customer_id" db:"customer_id" api:"The customer ID"`
	Type          string `json:"type" api:"The group AWS resource type"`
	toot          string
	Resource      *ec2.SecurityGroup `json:"resource" api:"The AWS resource data"`
	InstanceCount int                `json:"instance_count" db:"instance_count" api:"The number of instances in the group"`
	CreatedAt     time.Time          `json:"created_at" db:"created_at" api:"The group created time"`
	UpdatedAt     time.Time          `json:"updated_at" db:"updated_at" api:"The group updated time"`
	Nested        testNested
	NestedPtr     *testNested
}

func TestPlainStruct(t *testing.T) {
	t.SkipNow() // skipping this test until i fix the reflector

	groupType := GraphQLStructObject(testGroup{}, "Group", "The Opsee group check target")
	assert.Equal(t, "Group", groupType.Name())
	assert.Equal(t, "The Opsee group check target", groupType.PrivateDescription)

	fields := groupType.Fields()
	assert.Equal(t, "name", fields["name"].Name)
	assert.Equal(t, "String", fields["name"].Type.Name())
	assert.Equal(t, "The group identifier", fields["name"].Description)
	assert.Equal(t, "instance_count", fields["instance_count"].Name)
	assert.Equal(t, "Int", fields["instance_count"].Type.Name())
	assert.Equal(t, "The number of instances in the group", fields["instance_count"].Description)

	timeField := time.Now()

	tg := testGroup{
		Name:          "poo",
		CustomerID:    "whatever",
		Type:          "seccurity",
		toot:          "idk",
		InstanceCount: 5,
		CreatedAt:     timeField,
		Nested: testNested{
			Name:    "nested",
			IntData: 666,
		},
		NestedPtr: &testNested{
			Name:    "nestedPtr",
			IntData: 777,
			NamePtr: aws.String("nestedPtrNamePtr"),
		},
		Resource: &ec2.SecurityGroup{
			Description: aws.String("my dope SG"),
			GroupId:     aws.String("sg-dopedope"),
			GroupName:   aws.String("dope"),
			// Tags: []*ec2.Tag{
			// 	{
			// 		Key:   aws.String("opsee"),
			// 		Value: aws.String("is-dope"),
			// 	},
			// },
		},
	}

	t.Logf("%#v\n", groupType.Fields())
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"group": &graphql.Field{
					Type: groupType,
					Args: graphql.FieldConfigArgument{
						"id": &graphql.ArgumentConfig{
							Description: "The id of the group.",
							Type:        graphql.NewNonNull(graphql.String),
						},
						"type": &graphql.ArgumentConfig{
							Description: "The type of the group.",
							Type:        graphql.NewNonNull(graphql.String),
						},
					},
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return tg, nil
					},
				},
			},
		}),
	})

	if err != nil {
		t.Fatal(err)
	}

	res := graphql.Do(graphql.Params{Schema: schema, RequestString: `query goodQuery {
		group(id: "poo", type: "security") {
			name
			customer_id
			type
			instance_count
			created_at
			Nested {
				name
				int_data
			}
			NestedPtr {
				name
				int_data
				name_ptr
			}
			resource {
				Description
				GroupId
				GroupName
			}
		}
	}`})
	assert.Equal(t, false, res.HasErrors())
	// t.Logf("%#v\n", res.Errors)
	assert.Equal(t, tg.Name, getProp(res.Data, "group", "name"))
	assert.Equal(t, tg.CustomerID, getProp(res.Data, "group", "customer_id"))
	assert.Equal(t, tg.Type, getProp(res.Data, "group", "type"))
	assert.Equal(t, tg.InstanceCount, getProp(res.Data, "group", "instance_count"))
	assert.Equal(t, timeField.String(), getProp(res.Data, "group", "created_at"))
	assert.Equal(t, tg.Nested.Name, getProp(res.Data, "group", "Nested", "name"))
	assert.Equal(t, tg.Nested.IntData, getProp(res.Data, "group", "Nested", "int_data"))
	assert.Equal(t, tg.NestedPtr.Name, getProp(res.Data, "group", "NestedPtr", "name"))
	assert.Equal(t, *tg.NestedPtr.NamePtr, getProp(res.Data, "group", "NestedPtr", "name_ptr"))
	assert.Equal(t, tg.NestedPtr.IntData, getProp(res.Data, "group", "NestedPtr", "int_data"))
	assert.Equal(t, tg.Resource.Description, getProp(res.Data, "group", "resource", "Description"))
	assert.Equal(t, tg.Resource.GroupId, getProp(res.Data, "group", "resource", "GroupId"))
	assert.Equal(t, tg.Resource.GroupName, getProp(res.Data, "group", "resource", "GroupName"))

	// accessing unexported fields should error
	res = graphql.Do(graphql.Params{Schema: schema, RequestString: `query shitQueryUnexportedField {
		group(id: "poo", type: "security") {
			name
			customer_id
			type
			instance_count
			toot
		}
	}`})
	assert.Equal(t, true, res.HasErrors())

	// accessing non-existent fields should error
	res = graphql.Do(graphql.Params{Schema: schema, RequestString: `query shitQueryNonExistentField {
		group(id: "poo", type: "security") {
			name
			customer_id
			type
			instance_count
			toot
		}
	}`})
	assert.Equal(t, true, res.HasErrors())
}

func getProp(i interface{}, path ...string) interface{} {
	cur := i

	for _, s := range path {
		c, ok := cur.(map[string]interface{})
		if !ok {
			return cur
		}
		cur = c[s]
	}

	return cur
}
