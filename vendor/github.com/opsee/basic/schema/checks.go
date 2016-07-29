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

// Bartnet gives us {"type_url": "", "value": {}} instead of value being a byte array.
// So we have to do this shits manual-style. You can type switch on the return value,
// because I am lazy af.
func UnmarshalCrappyCheckSpecAnyJSON(data []byte) (interface{}, error) {
	var jsonSpec map[string]interface{}
	if err := json.Unmarshal(data, &jsonSpec); err != nil {
		return nil, err
	}

	typeURL, ok := jsonSpec["type_url"].(string)
	if !ok {
		return nil, fmt.Errorf("Unable to read type url: %v", jsonSpec["typeUrl"])
	}

	v, ok := jsonSpec["value"]
	if !ok {
		return nil, fmt.Errorf("No value in check spec json: %v", jsonSpec)
	}
	value, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Unable to convert value into map: %v", v)
	}

	switch typeURL {
	case "HttpCheck":
		/*
			message HttpCheck {
				string name = 1; //deprecated
				string path = 2 [(opseeproto.required) = true];
				string protocol = 3 [(opseeproto.required) = true];
				int32 port = 4 [(opseeproto.required) = true];
				string verb = 5 [(opseeproto.required) = true];
				repeated Header headers = 6;
				string body = 7;
			}

			message Header {
				string name = 1 [(opseeproto.required) = true];
				repeated string values = 2;
			}
		*/
		httpCheck := &HttpCheck{}
		path, ok := value["path"]
		if !ok {
			return nil, fmt.Errorf("Invalid HttpCheck spec, no path field")
		}
		pathStr, ok := path.(string)
		if !ok {
			return nil, fmt.Errorf("Invalid HttpCheck spec, path field not a string: %v", path)
		}
		httpCheck.Path = pathStr

		protocol, ok := value["protocol"]
		if !ok {
			return nil, fmt.Errorf("Invalid HttpCheck spec, no protocol field")
		}
		protocolStr, ok := protocol.(string)
		if !ok {
			return nil, fmt.Errorf("Invalid HttpCheck spec, protocol field not a string: %v", protocol)
		}
		httpCheck.Protocol = protocolStr

		port, ok := value["port"]
		if !ok {
			return nil, fmt.Errorf("Invalid HttpCheck spec, no port field")
		}
		portInt, ok := port.(float64)
		if !ok {
			return nil, fmt.Errorf("Invalid HttpCheck spec, port field not an int: %v", port)
		}
		httpCheck.Port = int32(portInt)

		verb, ok := value["verb"]
		if !ok {
			return nil, fmt.Errorf("Invalid HttpCheck spec, no verb field")
		}
		verbStr, ok := verb.(string)
		if !ok {
			return nil, fmt.Errorf("Invalid HttpCheck spec, verb field not a string: %v", verb)
		}
		httpCheck.Verb = verbStr

		body, ok := value["body"]
		if ok {
			bodyStr, ok := body.(string)
			if !ok {
				return nil, fmt.Errorf("Invalid HttpCheck spec, body field not a string: %v", body)
			}
			httpCheck.Body = bodyStr
		}

		headers, ok := value["headers"]
		if ok {
			headersArr, ok := headers.([]interface{})
			if !ok {
				return nil, fmt.Errorf("Invalid HttpCheck spec, header field invalid: %v", headers)
			}
			var hdrs []*Header
			for _, h := range headersArr {
				hmap := h.(map[string]interface{})
				name, ok := hmap["name"]
				if !ok {
					return nil, fmt.Errorf("Invalid HttpCheck spec, Header without a name: %v", h)
				}
				nameStr, ok := name.(string)
				if !ok {
					return nil, fmt.Errorf("Invalid HttpCheck spec, Header name field not a string: %v", name)
				}

				var valuesArr []string
				values, ok := hmap["values"]
				if ok {
					valuesArr, ok = values.([]string)
					if !ok {
						return nil, fmt.Errorf("Invalid HttpCheck spec, Header values field not a []string: %v", values)
					}
				}

				hdr := &Header{
					Name:   nameStr,
					Values: valuesArr,
				}
				hdrs = append(hdrs, hdr)
			}
			httpCheck.Headers = hdrs
		}

		return httpCheck, nil
	case "CloudWatchCheck":
		/*
			message CloudWatchCheck {
				repeated CloudWatchMetric metrics = 1;
			}

			message CloudWatchMetric {
				string namespace = 1;
				string name = 2;
			}
		*/
		cloudwatchCheck := &CloudWatchCheck{}
		metrics, ok := value["metrics"]
		if !ok {
			return nil, fmt.Errorf("CloudWatchCheck has no metrics field: %v", value)
		}

		metricsArr, ok := metrics.([]interface{})
		if !ok {
			return nil, fmt.Errorf("Invalid CloudWatchCheck spec, unable to convert metrics into an array: %v", metrics)
		}

		var cwMetrics []*CloudWatchMetric
		for _, m := range metricsArr {
			mMap, ok := m.(map[string]string)
			if !ok {
				return nil, fmt.Errorf("Invalid CloudWatchCheck spec, unable to convert array entry to map: %v", m)
			}

			namespace, ok := mMap["namespace"]
			if !ok {
				return nil, fmt.Errorf("Invalid CloudWatchCheck spec, metric found without namespace: %v", mMap)
			}

			name, ok := mMap["name"]
			if !ok {
				return nil, fmt.Errorf("Invalid CloudWatchCheck spec, metric missing name: %v", mMap)
			}

			metric := &CloudWatchMetric{
				Namespace: namespace,
				Name:      name,
			}
			cwMetrics = append(cwMetrics, metric)
		}
		cloudwatchCheck.Metrics = cwMetrics
		return cloudwatchCheck, nil
	}

	return nil, fmt.Errorf("Unknown check type in type URL: %q", typeURL)
}
