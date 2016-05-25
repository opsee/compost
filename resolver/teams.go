package resolver

import (
	"time"

	"github.com/opsee/basic/schema"
	opsee "github.com/opsee/basic/service"
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func (c *Client) GetTeam(ctx context.Context, user *schema.User) (*schema.Team, error) {
	log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	}).Info("get team request")

	req := &opsee.GetTeamRequest{
		Requestor: user,
		Team: &schema.Team{
			Id: user.CustomerId,
		},
	}
	resp, err := c.Vape.GetTeam(ctx, req)
	if err != nil {
		log.WithError(err).Error("error getting team from vape")
		return nil, err
	}

	return resp.Team, nil
}

func (c *Client) PutTeam(ctx context.Context, user *schema.User, teamInput map[string]interface{}) (*schema.Team, error) {
	log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	}).Info("put team request")

	// FIXME(mark)
	date1 := &opsee_types.Timestamp{}
	date2 := &opsee_types.Timestamp{}
	date1.Scan(time.Now().UTC().Add(-2 * 24 * 30 * time.Hour))
	date2.Scan(time.Now().UTC().Add(-1 * 24 * 30 * time.Hour))

	resp := &schema.Team{
		Id:           "deadbeef",
		Name:         "SeaOrg",
		Subscription: "basic",
		CreditCardInfo: &schema.CreditCardInfo{
			ExpMonth: int32(4),
			ExpYear:  int32(1992),
			Last4:    "6969",
			Brand:    "visa",
		},
		Users: []*schema.User{
			{
				Id:    7,
				Name:  "Tom Cruise",
				Email: "tom@cruise.com",
			},
			{
				Id:    8,
				Name:  "Isaac Johannsen",
				Email: "izs@cruise.com",
			},
		},
		Invoices: []*schema.Invoice{
			{
				Date:   date1,
				Amount: int32(6699),
			},
			{
				Date:   date2,
				Amount: int32(6699),
			},
		},
	}

	return resp, nil
}
