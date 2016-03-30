// Package bartnet provides a client interface to bartnet
package bartnet

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/opsee/basic/schema"
	"github.com/opsee/basic/service"
	"io"
	"io/ioutil"
	"net/http"
)

type Client interface {
	ListChecks(user *schema.User) ([]*schema.Check, error)
	CreateCheck(user *schema.User, check *schema.Check) (map[string]interface{}, error)
}

type client struct {
	client   *http.Client
	endpoint string
}

// An endpoint is the address of the bartnet service.
func New(endpoint string) Client {
	return &client{
		client:   &http.Client{},
		endpoint: endpoint,
	}
}

// ListChecks lists the checks + assertions for a user's customer account, without the results
func (c *client) ListChecks(user *schema.User) ([]*schema.Check, error) {
	body, err := c.do(user, "GET", "application/x-protobuf", "/gql/checks", nil)
	if err != nil {
		return nil, err
	}

	checks := &service.CheckResourceRequest{}
	err = proto.Unmarshal(body, checks)
	if err != nil {
		return nil, err
	}

	return checks.Checks, nil
}

func (c *client) CreateCheck(user *schema.User, check *schema.Check) (map[string]interface{}, error) {
	jsondata, err := marshalCheck(check)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(user, "POST", "application/json", "/checks", bytes.NewBuffer(jsondata))

	if err != nil {
		return nil, err
	}

	checkResp := make(map[string]interface{})
	err = json.Unmarshal(resp, &checkResp)
	if err != nil {
		return nil, err
	}

	return checkResp, nil
}

func (c *client) do(user *schema.User, method, accept, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, c.endpoint+path, body)
	if err != nil {
		return nil, err
	}

	toke, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString(toke)))
	req.Header.Set("Content-Type", accept)
	req.Header.Set("Accept", accept)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("bartnet responded with error status: %s", resp.Status)
	}

	return ioutil.ReadAll(resp.Body)
}

func marshalCheck(check *schema.Check) ([]byte, error) {
	httpCheckInt, err := schema.UnmarshalAny(check.CheckSpec)
	if err != nil {
		return nil, err
	}

	httpCheck, ok := httpCheckInt.(*schema.HttpCheck)
	if !ok {
		fmt.Errorf("couldn't unmarshal httpcheck")
	}

	jsonHttpCheck, err := json.Marshal(httpCheck)
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
		`{"name": "%s", "interval": 30, "target": %s, "check_spec": {"type_url": "HttpCheck", "value": %s}, "assertions": %s}`,
		check.Name,
		jsonTarget,
		jsonHttpCheck,
		jsonAssertions,
	)

	return []byte(jsonString), nil
}
