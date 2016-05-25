package resolver

import (
	"encoding/json"

	"github.com/opsee/basic/schema"
	opsee "github.com/opsee/basic/service"
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

	var team schema.Team
	tb, err := json.Marshal(teamInput)
	if err != nil {
		log.WithError(err).Error("marshal team input")
	}

	err = json.Unmarshal(tb, &team)
	if err != nil {
		log.WithError(err).Error("marshal team input")
	}

	req := &opsee.UpdateTeamRequest{
		Requestor: user,
		Team: &schema.Team{
			Id:           user.CustomerId,
			Name:         team.Name,
			Subscription: team.Subscription,
		},
	}

	resp, err := c.Vape.UpdateTeam(ctx, req)
	if err != nil {
		log.WithError(err).Error("error getting team from vape")
		return nil, err
	}

	return resp.Team, nil
}
