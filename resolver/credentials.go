package resolver

import (
	opsee "github.com/opsee/basic/service"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func (c *Client) GetCredentials(ctx context.Context, customerId string) (*opsee.GetCredentialsResponse, error) {
	log.WithFields(log.Fields{
		"customer_id": customerId,
	}).Info("get credentials request")

	userResp, err := c.Vape.GetUser(ctx, &opsee.GetUserRequest{CustomerId: customerId})
	if err != nil {
		log.WithError(err).Error("error getting user from vape")
		return nil, err
	}

	resp, err := c.Spanx.GetCredentials(ctx, &opsee.GetCredentialsRequest{userResp.User})
	if err != nil {
		log.WithError(err).Error("error getting credentials from spanx")
		return nil, err
	}

	return resp, nil
}
