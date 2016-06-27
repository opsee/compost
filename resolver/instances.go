package resolver

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/opsee/basic/schema"
	opsee_aws_ec2 "github.com/opsee/basic/schema/aws/ec2"
	opsee_aws_rds "github.com/opsee/basic/schema/aws/rds"
	opsee "github.com/opsee/basic/service"
	log "github.com/opsee/logrus"
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
	input := &opsee_aws_ec2.DescribeInstancesInput{
		Filters: []*opsee_aws_ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpc},
			},
		},
	}

	if instanceId != "" {
		input.InstanceIds = []string{instanceId}
	}

	resp, err := c.Bezos.Get(ctx, &opsee.BezosRequest{User: user, Region: region, VpcId: vpc, Input: &opsee.BezosRequest_Ec2_DescribeInstancesInput{input}})
	if err != nil {
		return nil, err
	}

	output := resp.GetEc2_DescribeInstancesOutput()
	if output == nil {
		return nil, fmt.Errorf("error decoding aws response")
	}

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
	// filter is not supported
	input := &opsee_aws_rds.DescribeDBInstancesInput{}

	if instanceId != "" {
		input.DBInstanceIdentifier = aws.String(instanceId)
	}

	resp, err := c.Bezos.Get(ctx, &opsee.BezosRequest{User: user, Region: region, VpcId: vpc, Input: &opsee.BezosRequest_Rds_DescribeDBInstancesInput{input}})
	if err != nil {
		return nil, err
	}

	output := resp.GetRds_DescribeDBInstancesOutput()
	if output == nil {
		return nil, fmt.Errorf("error decoding aws response")
	}

	return output.DBInstances, nil
}
