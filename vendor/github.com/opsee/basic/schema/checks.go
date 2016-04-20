package schema

import (
	"encoding/json"
	"fmt"
	"reflect"

	opsee_types "github.com/opsee/protobuf/opseeproto/types"
)

// register types
func init() {
	opsee_types.AnyTypeRegistry.RegisterAny("CloudWatchCheck", reflect.TypeOf(CloudWatchCheck{}))
	opsee_types.AnyTypeRegistry.RegisterAny("CloudWatchResponse", reflect.TypeOf(CloudWatchResponse{}))
	opsee_types.AnyTypeRegistry.RegisterAny("HttpCheck", reflect.TypeOf(HttpCheck{}))
	opsee_types.AnyTypeRegistry.RegisterAny("HttpResponse", reflect.TypeOf(HttpResponse{}))
}

// these exist because bartnet and beavis expect {typeurl: "blah", value: "blach"} in their json API
func (check *Check) MarshalCrappyJSON() ([]byte, error) {
	anySpec, err := opsee_types.UnmarshalAny(check.CheckSpec)
	if err != nil {
		return nil, err
	}

	jsonSpec, err := json.Marshal(anySpec)
	if err != nil {
		return nil, err
	}

	jsonTarget, err := json.Marshal(check.Target)
	if err != nil {
		return nil, err
	}

	jsonAssertions, err := json.Marshal(check.Assertions)
	if err != nil {
		return nil, err
	}

	jsonString := fmt.Sprintf(
		`{"name": "%s", "interval": 30, "target": %s, "check_spec": {"type_url": "%s", "value": %s}, "assertions": %s}`,
		check.Name,
		jsonTarget,
		check.CheckSpec.TypeUrl,
		jsonSpec,
		jsonAssertions,
	)

	return []byte(jsonString), nil
}
