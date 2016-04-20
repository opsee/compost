package hugs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/opsee/basic/schema"
	"net/http"
)

// this file is mainly bs!!!!!!
// it's a shim, and these objects should be proto, and we should have
// a proper service client. until hugs is fixed, i'm doing this to get launched
// also... why aren't notifications just part of checks???
type Client interface {
	CreateNotifications(user *schema.User, noteReq *NotificationRequest) error
	CreateNotificationsMulti(user *schema.User, noteReq []*NotificationRequest) error
}

type Notification struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type NotificationRequest struct {
	CheckId       string          `json:"check-id"`
	Notifications []*Notification `json:"notifications,array"`
}

type hugsClient struct {
	client   *http.Client
	endpoint string
}

func New(endpoint string) *hugsClient {
	return &hugsClient{
		client:   &http.Client{},
		endpoint: endpoint,
	}
}

func (c *hugsClient) CreateNotifications(user *schema.User, noteReq *NotificationRequest) error {
	return c.createNotifications(user, "/notifications", noteReq)
}

func (c *hugsClient) CreateNotificationsMulti(user *schema.User, noteReq []*NotificationRequest) error {
	return c.createNotifications(user, "/notifications-multicheck", noteReq)
}

func (c *hugsClient) createNotifications(user *schema.User, path string, noteReq interface{}) error {
	reqBody, err := json.Marshal(noteReq)
	if err != nil {
		return err
	}

	toke, err := json.Marshal(user)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.endpoint+path, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString(toke)))
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("hugs responded with error status: %s", resp.Status)
	}

	return nil
}
