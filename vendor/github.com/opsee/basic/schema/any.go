package schema

import (
	"fmt"
	"reflect"

	proto "github.com/gogo/protobuf/proto"
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
)

var (
	registry = make(map[string]reflect.Type)
)

func init() {
	// Check types for Any recomposition go here.
	registry["HttpCheck"] = reflect.TypeOf(HttpCheck{})
	registry["HttpResponse"] = reflect.TypeOf(HttpResponse{})
	registry["CloudWatchCheck"] = reflect.TypeOf(CloudWatchCheck{})
	registry["CloudWatchResponse"] = reflect.TypeOf(CloudWatchResponse{})
}

// UnmarshalAny unmarshals an Any object based on its TypeUrl type hint.

func UnmarshalAny(any *opsee_types.Any) (interface{}, error) {
	class := any.TypeUrl
	bytes := any.Value

	instance := reflect.New(registry[class]).Interface()
	err := proto.Unmarshal(bytes, instance.(proto.Message))
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// MarshalAny uses reflection to marshal an interface{} into an Any object and
// sets up its TypeUrl type hint.

func MarshalAny(i interface{}) (*opsee_types.Any, error) {
	msg, ok := i.(proto.Message)
	if !ok {
		err := fmt.Errorf("Unable to convert to proto.Message: %v", i)
		return nil, err
	}
	bytes, err := proto.Marshal(msg)

	if err != nil {
		return nil, err
	}

	return &opsee_types.Any{
		TypeUrl: reflect.ValueOf(i).Elem().Type().Name(),
		Value:   bytes,
	}, nil
}
