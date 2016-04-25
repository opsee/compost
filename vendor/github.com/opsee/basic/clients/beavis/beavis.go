// Package beavis provides a client interface to beavis
package beavis

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/opsee/basic/schema"
	"github.com/opsee/basic/service"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Client interface {
	ListResults(user *schema.User) ([]*schema.CheckResult, error)
	ListResultsCheck(user *schema.User, checkId string) ([]*schema.CheckResult, error)
	ListResultsTarget(user *schema.User, targetId string) ([]*schema.CheckResult, error)
}

type client struct {
	client   *http.Client
	endpoint string
}

// An endpoint is the address of the beavis service.
func New(endpoint string) Client {
	return &client{
		client:   &http.Client{},
		endpoint: endpoint,
	}
}

// ListResults lists all results for an account
func (c *client) ListResults(user *schema.User) ([]*schema.CheckResult, error) {
	return c.listResults(user, fmt.Sprintf("customer_id = \"%s\" and type = \"result\"", user.CustomerId))
}

// ListResults lists all results for an account
func (c *client) ListResultsCheck(user *schema.User, checkId string) ([]*schema.CheckResult, error) {
	return c.listResults(user, fmt.Sprintf("customer_id = \"%s\" and type = \"result\" and service = \"%s\"", user.CustomerId, checkId))
}

// ListResults lists all results for an account
func (c *client) ListResultsTarget(user *schema.User, targetId string) ([]*schema.CheckResult, error) {
	return c.listResults(user, fmt.Sprintf("customer_id = \"%s\" and type = \"result\" and host = \"%s\"", user.CustomerId, targetId))
}

func (c *client) listResults(user *schema.User, query string) ([]*schema.CheckResult, error) {
	body, err := c.do(user, "GET", "/gql/results?q="+url.QueryEscape(query), nil)
	if err != nil {
		return nil, err
	}

	results := &service.ResultsResource{}
	err = proto.Unmarshal(body, results)
	if err != nil {
		return nil, err
	}

	return results.Results, nil
}

func (c *client) do(user *schema.User, method, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, c.endpoint+path, body)
	if err != nil {
		return nil, err
	}

	toke, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString(toke)))
	req.Header.Set("Accept", "application/x-protobuf")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("beavis responded with error status: %s", resp.Status)
	}

	return ioutil.ReadAll(resp.Body)
}
