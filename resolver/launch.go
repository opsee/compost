package resolver

import (
	"github.com/opsee/basic/schema"
	opsee "github.com/opsee/basic/service"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func (c *Client) LaunchRoleUrlTemplate(ctx context.Context, user *schema.User) (string, error) {
	logger := log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	})
	logger.Info("launch role url template request")

	spanxResp, err := c.Spanx.EnhancedCombatMode(ctx, &opsee.EnhancedCombatModeRequest{
		User: user,
	})

	if err != nil {
		logger.WithError(err).Error("error getting new role url template")
		return "", err
	}

	logger.Infof("got new role url template: %s", spanxResp.StackUrl)
	return spanxResp.StackUrl, nil
}
