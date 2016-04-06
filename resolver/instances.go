package resolver

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/opsee/basic/schema"
	opsee_aws "github.com/opsee/basic/schema/aws"
	opsee_aws_ec2 "github.com/opsee/basic/schema/aws/ec2"
	opsee_aws_rds "github.com/opsee/basic/schema/aws/rds"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func (c *Client) GetInstances(ctx context.Context, user *schema.User, region, vpc, instanceType, instanceId string) (interface{}, error) {
	log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
	}).Info("get instances request")

	switch instanceType {
	case "ec2":
		return c.getInstancesEc2(ctx, user, region, vpc, instanceId)
	case "rds":
		return c.getInstancesRds(ctx, user, region, vpc, instanceId)
	}

	return fmt.Errorf("instance type not known: %s", instanceType), nil
}

func (c *Client) getInstancesEc2(ctx context.Context, user *schema.User, region, vpc, instanceId string) ([]*opsee_aws_ec2.Instance, error) {
	sess, err := c.awsSession(ctx, user, region)
	if err != nil {
		return nil, err
	}

	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpc)},
			},
		},
	}

	if instanceId != "" {
		input.InstanceIds = []*string{aws.String(instanceId)}
	}

	out, err := ec2.New(sess).DescribeInstances(input)
	if err != nil {
		return nil, err
	}

	output := &opsee_aws_ec2.DescribeInstancesOutput{}
	opsee_aws.CopyInto(output, out)

	instances := make([]*opsee_aws_ec2.Instance, 0)
	for _, res := range output.Reservations {
		if res.Instances == nil {
			continue
		}

		for _, inst := range res.Instances {
			instances = append(instances, inst)
		}
	}

	return instances, nil
}

func (c *Client) getInstancesRds(ctx context.Context, user *schema.User, region, vpc, instanceId string) ([]*opsee_aws_rds.DBInstance, error) {
	sess, err := c.awsSession(ctx, user, region)
	if err != nil {
		return nil, err
	}

	// filter is not supported
	input := &rds.DescribeDBInstancesInput{}

	if instanceId != "" {
		input.DBInstanceIdentifier = aws.String(instanceId)
	}

	out, err := rds.New(sess).DescribeDBInstances(input)
	if err != nil {
		return nil, err
	}

	output := &opsee_aws_rds.DescribeDBInstancesOutput{}
	opsee_aws.CopyInto(output, out)

	return output.DBInstances, nil
}
