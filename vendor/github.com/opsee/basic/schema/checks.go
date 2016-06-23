package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/gogo/protobuf/jsonpb"
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
)

// register types
func init() {
	opsee_types.AnyTypeRegistry.Register("CloudWatchCheck", reflect.TypeOf(CloudWatchCheck{}))
	opsee_types.AnyTypeRegistry.Register("CloudWatchResponse", reflect.TypeOf(CloudWatchResponse{}))
	opsee_types.AnyTypeRegistry.Register("HttpCheck", reflect.TypeOf(HttpCheck{}))
	opsee_types.AnyTypeRegistry.Register("HttpResponse", reflect.TypeOf(HttpResponse{}))
}

// CheckResponseReply is the exported version of isCheckResponse_Reply
// which allows us to use that interface and make CheckResponse.Reply
// easier to set indirectly in bastion workers.
type CheckResponseReply isCheckResponse_Reply

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

var TargetTypes = []string{"dbinstance", "instance", "asg", "sg", "elb", "host", "ecs_service"}

func (t *Target) Validate() error {
	i := sort.Search(len(TargetTypes), func(i int) bool { return TargetTypes[i] == t.Type })
	if i < len(TargetTypes) && TargetTypes[i] == t.Type {
		return nil
	} else {
		return fmt.Errorf("Invalid target type: %s", t.Type)
	}
}

// Validate a Check and ensure it has required fields.
func (c *Check) Validate() error {
	if c.Id == "" {
		return fmt.Errorf("Check missing ID.")
	}

	if c.CustomerId == "" {
		return fmt.Errorf("Check missing Customer ID.")
	}

	if c.Target == nil {
		return fmt.Errorf("Check missing Target.")
	}

	if c.Spec == nil {
		return fmt.Errorf("Check missing Spec.")
	}

	if err := c.Target.Validate(); err != nil {
		return err
	}

	return nil
}

// Checks need their any fields
func (c *Check) MarshalJSON() ([]byte, error) {
	var jsonBytes bytes.Buffer

	// marshal to json using jsonpb
	marshaler := jsonpb.Marshaler{
		EmitDefaults: true,
	}

	err := marshaler.Marshal(&jsonBytes, c)
	if err != nil {
		return nil, err
	}

	return jsonBytes.Bytes(), nil
}

func (c *Check) UnmarshalJSON(data []byte) error {
	return jsonpb.Unmarshal(bytes.NewBuffer(data), c)
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
		`{"name": "%s", "min_failing_count": %d, "min_failing_time": %d, "interval": 30, "target": %s, "check_spec": {"type_url": "%s", "value": %s}, "assertions": %s}`,
		check.Name,
		check.MinFailingCount,
		check.MinFailingTime,
		jsonTarget,
		typeUrl,
		jsonSpec,
		jsonAssertions,
	)

	return []byte(jsonString), nil
}
