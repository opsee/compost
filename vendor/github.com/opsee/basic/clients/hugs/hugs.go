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
	ListNotifications(user *schema.User) ([]*Notification, error)
	ListNotificationsDefault(user *schema.User) ([]*Notification, error)
	ListNotificationsCheck(user *schema.User, checkId string) ([]*Notification, error)
	CreateNotifications(user *schema.User, noteReq *NotificationRequest) error
	CreateNotificationsDefault(user *schema.User, noteReq *NotificationRequest) error
	CreateNotificationsMulti(user *schema.User, noteReq []*NotificationRequest) error
}

type Notification struct {
	CheckId string `json:"check_id"`
	Type    string `json:"type"`
	Value   string `json:"value"`
}

type NotificationResponse struct {
	Notifications []*Notification `json:"notifications"`
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

func (c *hugsClient) ListNotifications(user *schema.User) ([]*Notification, error) {
	return c.listNotifications(user, "/notifications")
}

func (c *hugsClient) ListNotificationsDefault(user *schema.User) ([]*Notification, error) {
	return c.listNotifications(user, "/notifications-default")
}

func (c *hugsClient) ListNotificationsCheck(user *schema.User, checkId string) ([]*Notification, error) {
	return c.listNotifications(user, fmt.Sprintf("/notifications/%s", checkId))
}

func (c *hugsClient) listNotifications(user *schema.User, path string) ([]*Notification, error) {
	toke, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", c.endpoint+path, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString(toke)))
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hugs responded with error status: %s", resp.Status)
	}

	var notifications *NotificationResponse
	err = json.NewDecoder(resp.Body).Decode(&notifications)
	if err != nil {
		return nil, err
	}

	return notifications.Notifications, nil
}

func (c *hugsClient) CreateNotifications(user *schema.User, noteReq *NotificationRequest) error {
	return c.createNotifications(user, "/notifications", noteReq)
}

func (c *hugsClient) CreateNotificationsDefault(user *schema.User, noteReq *NotificationRequest) error {
	return c.createNotifications(user, "/notifications-default", noteReq)
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
