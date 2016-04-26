// Package bartnet provides a client interface to bartnet
package bartnet

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/opsee/basic/schema"
	"github.com/opsee/basic/service"
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
)

type Client interface {
	GetCheck(user *schema.User, id string) (*schema.Check, error)
	ListChecks(user *schema.User) ([]*schema.Check, error)
	CreateCheck(user *schema.User, check *schema.Check) (*schema.Check, error)
	UpdateCheck(user *schema.User, check *schema.Check) (*schema.Check, error)
	DeleteCheck(user *schema.User, id string) error
	TestCheck(user *schema.User, check *schema.Check, deadline time.Time, maxHosts int) (*service.TestCheckResponse, error)
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

// GetCheck gets a check + assertions, without the results
func (c *client) GetCheck(user *schema.User, id string) (*schema.Check, error) {
	if id == "" {
		return nil, fmt.Errorf("can't get check without an id")
	}

	body, err := c.do(user, "GET", "application/x-protobuf", fmt.Sprintf("/gql/checks/%s", id), nil)
	if err != nil {
		return nil, err
	}

	check := &schema.Check{}
	err = proto.Unmarshal(body, check)
	if err != nil {
		return nil, err
	}

	return check, nil
}

func (c *client) TestCheck(user *schema.User, check *schema.Check, deadline time.Time, maxHosts int) (*service.TestCheckResponse, error) {
	var ts *opsee_types.Timestamp
	if err := ts.Scan(deadline); err != nil {
		return nil, err
	}

	tsJson, err := ts.MarshalJSON()
	if err != nil {
		return nil, err
	}

	checkJson, err := check.MarshalCrappyJSON()
	if err != nil {
		return nil, err
	}

	testCheckJson := fmt.Sprintf(`{"max_hosts": %d, "deadline": %s, "check": %s}`, maxHosts, string(tsJson), string(checkJson))

	body, err := c.do(user, "POST", "application/x-protobuf", "/bastions/test-check", bytes.NewBufferString(testCheckJson))
	if err != nil {
		return nil, err
	}

	testCheckResp := &service.TestCheckResponse{}
	err = proto.Unmarshal(body, testCheckResp)
	if err != nil {
		return nil, err
	}

	return testCheckResp, nil
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

func (c *client) CreateCheck(user *schema.User, check *schema.Check) (*schema.Check, error) {
	jsondata, err := check.MarshalCrappyJSON()
	if err != nil {
		return nil, err
	}

	resp, err := c.do(user, "POST", "application/x-protobuf", "/checks", bytes.NewBuffer(jsondata))

	if err != nil {
		return nil, err
	}

	checkResponse := &service.CheckResourceRequest{}
	err = proto.Unmarshal(resp, checkResponse)
	if err != nil {
		return nil, err
	}

	if len(checkResponse.Checks) < 1 {
		return nil, fmt.Errorf("no checks returned")
	}

	return checkResponse.Checks[0], nil
}

func (c *client) UpdateCheck(user *schema.User, check *schema.Check) (*schema.Check, error) {
	if check.Id == "" {
		return nil, fmt.Errorf("can't update check without an id")
	}

	jsondata, err := check.MarshalCrappyJSON()
	if err != nil {
		return nil, err
	}

	resp, err := c.do(user, "PUT", "application/x-protobuf", fmt.Sprintf("/checks/%s", check.Id), bytes.NewBuffer(jsondata))

	if err != nil {
		return nil, err
	}

	checkResponse := &schema.Check{}
	err = proto.Unmarshal(resp, checkResponse)
	if err != nil {
		return nil, err
	}

	return checkResponse, nil
}

func (c *client) DeleteCheck(user *schema.User, id string) error {
	if id == "" {
		return fmt.Errorf("can't delete a check without id")
	}

	_, err := c.do(user, "DELETE", "application/json", fmt.Sprintf("/checks/%s", id), nil)
	if err != nil {
		return err
	}

	return nil
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
	req.Header.Set("Content-Type", "application/json")
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
