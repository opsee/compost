package resolver

import (
	"github.com/opsee/basic/clients/hugs"
	"github.com/opsee/basic/schema"
	log "github.com/opsee/logrus"
	"golang.org/x/net/context"
)

func (c *Client) GetNotifications(ctx context.Context, user *schema.User, defaultOnly bool) ([]*schema.Notification, error) {
	logger := log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	})
	logger.Info("get notifications request")

	// we only have default notifications as first-class objects right now.
	// in the future, you will be able to list other notifications and update
	// the objects that point to them
	if defaultOnly {
		notifs, err := c.Hugs.ListNotificationsDefault(user)
		if err != nil {
			logger.WithError(err).Error("hugs error")
			return nil, err
		}

		// cheating here, i hate that the hugs client isn't grpc
		var result []*schema.Notification
		for _, r := range notifs {
			result = append(result, &schema.Notification{Type: r.Type, Value: r.Value})
		}

		return result, nil
	}

	return nil, nil
}

func (c *Client) PutDefaultNotifications(ctx context.Context, user *schema.User, notificationsInput []interface{}) ([]*schema.Notification, error) {
	logger := log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	})
	logger.Info("put notifications request")

	var notifs []*hugs.Notification
	for _, notif := range notificationsInput {
		if n, ok := notif.(map[string]interface{}); ok {
			value, _ := n["value"].(string)
			typ, _ := n["type"].(string)
			notifs = append(notifs, &hugs.Notification{Value: value, Type: typ})
		}
	}

	err := c.Hugs.CreateNotificationsDefault(user, &hugs.NotificationRequest{Notifications: notifs})
	if err != nil {
		logger.WithError(err).Error("hugs error")
		return nil, err
	}

	// cheating here, i hate that the hugs client isn't grpc
	var result []*schema.Notification
	for _, r := range notifs {
		result = append(result, &schema.Notification{Type: r.Type, Value: r.Value})
	}

	return result, nil
}
