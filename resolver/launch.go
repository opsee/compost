package resolver

import (
	"github.com/opsee/basic/schema"
	opsee "github.com/opsee/basic/service"
	log "github.com/opsee/logrus"
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
	if spanxResp.TemplateUrl != "" {
		return spanxResp.TemplateUrl, nil
	} else {
		return spanxResp.StackUrl, nil
	}
}

func (c *Client) LaunchRoleUrl(ctx context.Context, user *schema.User) (string, error) {
	logger := log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	})
	logger.Info("launch role url request")

	spanxResp, err := c.Spanx.EnhancedCombatMode(ctx, &opsee.EnhancedCombatModeRequest{
		User: user,
	})

	if err != nil {
		logger.WithError(err).Error("error getting new role url")
		return "", err
	}

	logger.Infof("got new role url template: %s", spanxResp.StackUrl)
	return spanxResp.StackUrl, nil
}

func (c *Client) HasRole(ctx context.Context, user *schema.User) (bool, error) {
	logger := log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	})
	logger.Info("has role request")

	_, err := c.Spanx.GetCredentials(ctx, &opsee.GetCredentialsRequest{
		User: user,
	})

	if err != nil {
		logger.WithError(err).Error("error getting spanx creds, may be normal")
		return false, nil
	}

	return true, nil
}

func (c *Client) GetRoleStack(ctx context.Context, user *schema.User) (*schema.RoleStack, error) {
	logger := log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	})
	logger.Info("role request")

	resp, err := c.Spanx.GetRoleStack(ctx, &opsee.GetRoleStackRequest{
		User: user,
	})

	if err != nil {
		logger.WithError(err).Error("error getting role stack")
		return nil, err
	}

	if resp == nil {
		logger.Debug("no role stack found")
		return nil, nil
	}

	return resp.RoleStack, nil
}

func (c *Client) ScanRegion(ctx context.Context, user *schema.User, region string) (*schema.Region, error) {
	logger := log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	})
	logger.Info("scan region request")

	resp, err := c.Keelhaul.ScanVpcs(ctx, &opsee.ScanVpcsRequest{
		User:   user,
		Region: region,
	})

	if err != nil {
		logger.WithError(err).Error("error scanning region")
		return nil, err
	}

	logger.Infof("scanned region: %s", region)
	return resp.Region, nil
}

func (c *Client) LaunchBastionStack(ctx context.Context, user *schema.User, region, vpcId, subnetId, subnetRouting, instanceSize string) (bool, error) {
	logger := log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	})
	logger.Info("launch bastion stack request")

	_, err := c.Keelhaul.LaunchStack(ctx, &opsee.LaunchStackRequest{
		User:          user,
		Region:        region,
		VpcId:         vpcId,
		SubnetId:      subnetId,
		SubnetRouting: subnetRouting,
		InstanceSize:  instanceSize,
	})

	if err != nil {
		logger.WithError(err).Error("error launching bastion stack")
		return false, err
	}

	logger.Infof("launched stack - region: %s, vpc: %s, subnet: %s, routing: %s, size: %s", region, vpcId, subnetId, subnetRouting, instanceSize)
	return true, nil
}
