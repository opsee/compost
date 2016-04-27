package resolver

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/opsee/basic/schema"
	opsee_aws_autoscaling "github.com/opsee/basic/schema/aws/autoscaling"
	opsee_aws_ec2 "github.com/opsee/basic/schema/aws/ec2"
	opsee_aws_elb "github.com/opsee/basic/schema/aws/elb"
	opsee "github.com/opsee/basic/service"
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
	case "autoscaling":
		return c.getGroupsAutoscaling(ctx, user, region, vpc, groupId)
	}

	return fmt.Errorf("group type not known: %s", groupType), nil
}

func (c *Client) getGroupsSecurity(ctx context.Context, user *schema.User, region, vpc, groupId string) ([]*opsee_aws_ec2.SecurityGroup, error) {
	input := &opsee_aws_ec2.DescribeSecurityGroupsInput{
		Filters: []*opsee_aws_ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpc},
			},
		},
	}

	if groupId != "" {
		input.GroupIds = []string{groupId}
	}

	resp, err := c.Bezos.Get(ctx, &opsee.BezosRequest{User: user, Region: region, VpcId: vpc, Input: &opsee.BezosRequest_Ec2_DescribeSecurityGroupsInput{input}})
	if err != nil {
		return nil, err
	}

	output := resp.GetEc2_DescribeSecurityGroupsOutput()
	if output == nil {
		return nil, fmt.Errorf("error decoding aws response")
	}

	return output.SecurityGroups, nil
}

func (c *Client) getGroupsElb(ctx context.Context, user *schema.User, region, vpc, groupId string) ([]*opsee_aws_elb.LoadBalancerDescription, error) {
	// filter is not supported
	input := &opsee_aws_elb.DescribeLoadBalancersInput{}

	if groupId != "" {
		input.LoadBalancerNames = []string{groupId}
	}

	resp, err := c.Bezos.Get(ctx, &opsee.BezosRequest{User: user, Region: region, VpcId: vpc, Input: &opsee.BezosRequest_Elb_DescribeLoadBalancersInput{input}})
	if err != nil {
		return nil, err
	}

	output := resp.GetElb_DescribeLoadBalancersOutput()
	if output == nil {
		return nil, fmt.Errorf("error decoding aws response")
	}

	return output.LoadBalancerDescriptions, nil
}

func (c *Client) getGroupsAutoscaling(ctx context.Context, user *schema.User, region, vpc, groupId string) ([]*opsee_aws_autoscaling.Group, error) {
	// filter is not supported
	input := &opsee_aws_autoscaling.DescribeAutoScalingGroupsInput{}

	if groupId != "" {
		input.AutoScalingGroupNames = []string{groupId}
	}

	resp, err := c.Bezos.Get(ctx, &opsee.BezosRequest{User: user, Region: region, VpcId: vpc, Input: &opsee.BezosRequest_Autoscaling_DescribeAutoScalingGroupsInput{input}})
	if err != nil {
		return nil, err
	}

	output := resp.GetAutoscaling_DescribeAutoScalingGroupsOutput()
	if output == nil {
		return nil, fmt.Errorf("error decoding aws response")
	}

	return output.AutoScalingGroups, nil
}
