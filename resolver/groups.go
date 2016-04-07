package resolver

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/opsee/basic/schema"
	opsee_aws "github.com/opsee/basic/schema/aws"
	opsee_aws_ec2 "github.com/opsee/basic/schema/aws/ec2"
	opsee_aws_elb "github.com/opsee/basic/schema/aws/elb"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func (c *Client) GetGroups(ctx context.Context, user *schema.User, region, vpc, groupType, groupId string) (interface{}, error) {
	log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
	}).Info("get groups request")

	switch groupType {
	case "security":
		return c.getGroupsSecurity(ctx, user, region, vpc, groupId)
	case "elb":
		return c.getGroupsElb(ctx, user, region, vpc, groupId)
	}

	return fmt.Errorf("group type not known: %s", groupType), nil
}

func (c *Client) getGroupsSecurity(ctx context.Context, user *schema.User, region, vpc, groupId string) ([]*opsee_aws_ec2.SecurityGroup, error) {
	sess, err := c.awsSession(ctx, user, region)
	if err != nil {
		return nil, err
	}

	input := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpc)},
			},
		},
	}

	if groupId != "" {
		input.GroupIds = []*string{aws.String(groupId)}
	}

	out, err := ec2.New(sess).DescribeSecurityGroups(input)
	if err != nil {
		return nil, err
	}

	output := &opsee_aws_ec2.DescribeSecurityGroupsOutput{}
	opsee_aws.CopyInto(output, out)

	return output.SecurityGroups, nil
}

func (c *Client) getGroupsElb(ctx context.Context, user *schema.User, region, vpc, groupId string) ([]*opsee_aws_elb.LoadBalancerDescription, error) {
	sess, err := c.awsSession(ctx, user, region)
	if err != nil {
		return nil, err
	}

	// filter is not supported
	input := &elb.DescribeLoadBalancersInput{}

	if groupId != "" {
		input.LoadBalancerNames = []*string{aws.String(groupId)}
	}

	out, err := elb.New(sess).DescribeLoadBalancers(input)
	if err != nil {
		return nil, err
	}

	output := &opsee_aws_elb.DescribeLoadBalancersOutput{}
	opsee_aws.CopyInto(output, out)

	return output.LoadBalancerDescriptions, nil
}
