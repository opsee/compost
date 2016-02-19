// Package spanx provides a client interface to the spanx service.
package spanx

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/opsee/basic/com"
	"io"
	"net/http"
)

type Client interface {
	PutRole(user *com.User, accessKey, secretKey string) (credentials.Value, error)
	GetCredentials(user *com.User) (credentials.Value, error)
}

type client struct {
	client   *http.Client
	endpoint string
}

type roleRequest struct {
	AccessKeyID     string
	SecretAccessKey string
}

type credentialsResponse struct {
	Credentials credentials.Value
}

// An endpoint is the address of the spanx service.
func New(endpoint string) Client {
	return &client{
		client:   &http.Client{},
		endpoint: endpoint,
	}
}

// PutRole provisions the Opsee IAM role in a customer's account, using the provided AWS credentials.
// Calls to PutRole are idempotent; if a role has already been provisioned, it will simply return
// AWS STS credentials.
func (c *client) PutRole(user *com.User, accessKey, secretKey string) (credentials.Value, error) {
	var creds credentials.Value

	body, err := json.Marshal(roleRequest{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	})

	if err != nil {
		return creds, err
	}

	return c.do(user, "PUT", "/credentials", bytes.NewBuffer(body))
}

// GetCredentials returns an AWS STS credentials value that expires at the configured expiration of
// the spanx service.
func (c *client) GetCredentials(user *com.User) (credentials.Value, error) {
	return c.do(user, "GET", "/credentials", nil)
}

func (c *client) do(user *com.User, method, path string, body io.Reader) (credentials.Value, error) {
	var creds credentials.Value

	req, err := http.NewRequest(method, c.endpoint+path, body)
	if err != nil {
		return creds, err
	}

	toke, err := json.Marshal(user)
	if err != nil {
		return creds, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString(toke)))

	resp, err := c.client.Do(req)
	if err != nil {
		return creds, err
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return creds, fmt.Errorf("spanx responded with error status: %s", resp.Status)
	}

	credsResponse := &credentialsResponse{}
	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(credsResponse)
	if err != nil {
		return creds, err
	}

	return credsResponse.Credentials, nil
}
