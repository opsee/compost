package resolver

import (
	"github.com/opsee/basic/schema"
	opsee "github.com/opsee/basic/service"
	log "github.com/opsee/logrus"
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

	resp, err := c.Cats.GetTeam(ctx, req)
	if err != nil {
		log.WithError(err).Error("error getting team from cats")
		return nil, err
	}

	var fu []*schema.User

	if resp.Team != nil {
		for _, u := range resp.Team.Users {
			fu = append(fu, &schema.User{Id: u.Id, Name: u.Name, Email: u.Email, Perms: u.Perms, Status: u.Status})
		}
	}
	resp.Team.Users = fu

	return resp.Team, nil
}

func (c *Client) PutTeam(ctx context.Context, user *schema.User, teamInput map[string]interface{}) (*schema.Team, error) {
	log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	}).Info("put team request")

	var (
		plan        string
		stripeToken string
		name        string
	)

	name, _ = teamInput["name"].(string)
	plan, _ = teamInput["plan"].(string)
	stripeToken, _ = teamInput["stripeToken"].(string)

	req := &opsee.UpdateTeamRequest{
		Requestor: user,
		Team: &schema.Team{
			Id:               user.CustomerId,
			Name:             name,
			SubscriptionPlan: plan,
		},
		StripeToken: stripeToken,
	}

	resp, err := c.Cats.UpdateTeam(ctx, req)
	if err != nil {
		log.WithError(err).Error("error updating team")
		return nil, err
	}

	return resp.Team, nil
}
