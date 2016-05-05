package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/gogo/protobuf/jsonpb"
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
)

// register types
func init() {
	opsee_types.AnyTypeRegistry.RegisterAny("CloudWatchCheck", reflect.TypeOf(CloudWatchCheck{}))
	opsee_types.AnyTypeRegistry.RegisterAny("CloudWatchResponse", reflect.TypeOf(CloudWatchResponse{}))
	opsee_types.AnyTypeRegistry.RegisterAny("HttpCheck", reflect.TypeOf(HttpCheck{}))
	opsee_types.AnyTypeRegistry.RegisterAny("HttpResponse", reflect.TypeOf(HttpResponse{}))
}

// Metrics need to include double 0 values so we must use jsonpb to marshal them to json.
func (m *Metric) MarshalJSON() ([]byte, error) {
	var jsonBytes bytes.Buffer

	// marshal to json using jsonpb
	marshaler := jsonpb.Marshaler{
		EmitDefaults: true,
	}

	err := marshaler.Marshal(&jsonBytes, m)
	if err != nil {
		return nil, err
	}

	return jsonBytes.Bytes(), nil
}

// Checks need their any fields
func (m *Check) MarshalJSON() ([]byte, error) {
	var jsonBytes bytes.Buffer

	// marshal to json using jsonpb
	marshaler := jsonpb.Marshaler{
		EmitDefaults: true,
	}

	err := marshaler.Marshal(&jsonBytes, m)
	if err != nil {
		return nil, err
	}

	return jsonBytes.Bytes(), nil
}

func (m *Check) UnmarshalJSON(data []byte) error {
	return jsonpb.Unmarshal(bytes.NewBuffer(data), m)
}

// these exist because bartnet and beavis expect {typeurl: "blah", value: "blach"} in their json API
func (check *Check) MarshalCrappyJSON() ([]byte, error) {
	var (
		anySpec interface{}
		typeUrl string
		err     error
	)

	// if no checkspec, copy over "Spec"
	if check.CheckSpec == nil {
		if check.Spec == nil {
			return nil, fmt.Errorf("check is missing Spec and CheckSpec")
		}

		switch t := check.Spec.(type) {
		case *Check_HttpCheck:
			anySpec = t.HttpCheck
			typeUrl = "HttpCheck"
		case *Check_CloudwatchCheck:
			anySpec = t.CloudwatchCheck
			typeUrl = "CloudWatchCheck"
		}
	} else {
		anySpec, err = opsee_types.UnmarshalAny(check.CheckSpec)
		if err != nil {
			return nil, err
		}

		typeUrl = check.CheckSpec.TypeUrl
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
		typeUrl,
		jsonSpec,
		jsonAssertions,
	)

	return []byte(jsonString), nil
}
