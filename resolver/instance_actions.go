package resolver

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/opsee/basic/schema"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func (c *Client) RebootInstances(ctx context.Context, user *schema.User, region string, instanceIds []string) error {
	logger := log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
	})
	logger.Info("reboot instances request")

	session, err := c.awsSession(ctx, user, region)
	if err != nil {
		logger.WithError(err).Error("error acquiring aws session")
		return err
	}

	_, err = ec2.New(session).RebootInstances(&ec2.RebootInstancesInput{
		InstanceIds: aws.StringSlice(instanceIds),
	})

	if err != nil {
		logger.WithError(err).Error("error rebooting instances")
		return err
	}

	return nil
}

func (c *Client) StartInstances(ctx context.Context, user *schema.User, region string, instanceIds []string) error {
	logger := log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
	})
	logger.Info("start instances request")

	session, err := c.awsSession(ctx, user, region)
	if err != nil {
		logger.WithError(err).Error("error acquiring aws session")
		return err
	}

	_, err = ec2.New(session).StartInstances(&ec2.StartInstancesInput{
		InstanceIds: aws.StringSlice(instanceIds),
	})

	if err != nil {
		logger.WithError(err).Error("error starting instances")
		return err
	}

	return nil
}

func (c *Client) StopInstances(ctx context.Context, user *schema.User, region string, instanceIds []string) error {
	logger := log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
	})
	logger.Info("stop instances request")

	session, err := c.awsSession(ctx, user, region)
	if err != nil {
		logger.WithError(err).Error("error acquiring aws session")
		return err
	}

	_, err = ec2.New(session).StopInstances(&ec2.StopInstancesInput{
		InstanceIds: aws.StringSlice(instanceIds),
	})

	if err != nil {
		logger.WithError(err).Error("error stopping instances")
		return err
	}

	return nil
}
