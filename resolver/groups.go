package resolver

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/opsee/basic/schema"
	opsee_aws_autoscaling "github.com/opsee/basic/schema/aws/autoscaling"
	opsee_aws_ec2 "github.com/opsee/basic/schema/aws/ec2"
	opsee_aws_ecs "github.com/opsee/basic/schema/aws/ecs"
	opsee_aws_elb "github.com/opsee/basic/schema/aws/elb"
	opsee "github.com/opsee/basic/service"
	log "github.com/opsee/logrus"
	"golang.org/x/net/context"
)

// Fetches a single task definition for ECS services
func (c *Client) GetTaskDefinition(ctx context.Context, user *schema.User, region, id string) (*opsee_aws_ecs.TaskDefinition, error) {
	log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
	}).Info("get task definition request")

	resp, err := c.Bezos.Get(
		ctx,
		&opsee.BezosRequest{
			User:   user,
			Region: region,
			VpcId:  "none",
			Input: &opsee.BezosRequest_Ecs_DescribeTaskDefinitionInput{
				&opsee_aws_ecs.DescribeTaskDefinitionInput{
					TaskDefinition: aws.String(id),
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	output := resp.GetEcs_DescribeTaskDefinitionOutput()
	if output == nil {
		return nil, fmt.Errorf("error decoding aws response")
	}

	return output.TaskDefinition, nil
}

func (c *Client) GetGroups(ctx context.Context, user *schema.User, region, vpc, groupType, groupId string) (interface{}, error) {
	log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
	}).Info("get groups request")

	switch groupType {
	case "security":
		return c.getGroupsSecurity(ctx, user, region, vpc, groupId)
	case "ecs_service":
		return c.getGroupsEcsService(ctx, user, region, vpc, groupId)
	case "elb":
		return c.getGroupsElb(ctx, user, region, vpc, groupId)
	case "autoscaling":
		return c.getGroupsAutoscaling(ctx, user, region, vpc, groupId)
	case "":
		groupsSec, err := c.getGroupsSecurity(ctx, user, region, vpc, groupId)
		if err != nil {
			return nil, err
		}

		groupsEcs, err := c.getGroupsEcsService(ctx, user, region, vpc, groupId)
		if err != nil {
			return nil, err
		}

		groupsElb, err := c.getGroupsElb(ctx, user, region, vpc, groupId)
		if err != nil {
			return nil, err
		}

		groupsAut, err := c.getGroupsAutoscaling(ctx, user, region, vpc, groupId)
		if err != nil {
			return nil, err
		}

		// TODO: concurrency
		var allgroups []interface{}
		for _, g := range groupsSec {
			allgroups = append(allgroups, interface{}(g))
		}
		for _, g := range groupsEcs {
			allgroups = append(allgroups, interface{}(g))
		}
		for _, g := range groupsElb {
			allgroups = append(allgroups, interface{}(g))
		}
		for _, g := range groupsAut {
			allgroups = append(allgroups, interface{}(g))
		}

		return allgroups, nil
	}

	return fmt.Errorf("group type not known: %s", groupType), nil
}

// getGroupsEcsService takes the normal arguments, but the groupId argument is actually a tuple of
// (ecs cluster name/arn, service name/arn). If left blank, we will try our best to find all of the
// services running only on clusters deployed to this VPC.
func (c *Client) getGroupsEcsService(ctx context.Context, user *schema.User, region, vpc, groupId string) ([]*opsee_aws_ecs.Service, error) {
	logger := log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"endpoint":    "getGroupsEcsService",
	})
	logger.Info("get groups request")

	if groupId != "" {
		t := strings.Split(groupId, "/")
		if len(t) < 2 {
			return nil, fmt.Errorf("Invalid group id for ECS Service")
		}

		cluster_id := t[0]
		service_name := t[1]

		input := &opsee_aws_ecs.DescribeServicesInput{
			Services: []string{service_name},
			Cluster:  aws.String(cluster_id),
		}

		resp, err := c.Bezos.Get(
			ctx,
			&opsee.BezosRequest{
				User:   user,
				Region: region,
				VpcId:  vpc,
				Input:  &opsee.BezosRequest_Ecs_DescribeServicesInput{input},
			},
		)
		if err != nil {
			return nil, err
		}

		output := resp.GetEcs_DescribeServicesOutput()
		if output == nil {
			return nil, fmt.Errorf("error decoding aws response")
		}

		if len(output.Services) == 0 {
			logger.Info("no services found")
		}

		return output.Services, nil
	}

	lcInput := &opsee_aws_ecs.ListClustersInput{}
	resp, err := c.Bezos.Get(
		ctx,
		&opsee.BezosRequest{
			User:   user,
			Region: region,
			VpcId:  vpc,
			Input:  &opsee.BezosRequest_Ecs_ListClustersInput{lcInput},
		},
	)
	if err != nil {
		return nil, err
	}

	// TODO(greg): support paging. god help us.
	lcOutput := resp.GetEcs_ListClustersOutput()
	if lcOutput == nil {
		return nil, fmt.Errorf("error decoding aws response")
	}

	if len(lcOutput.ClusterArns) == 0 {
		logger.Info("no clusters found")
	}

	var svcs []*opsee_aws_ecs.Service

	for _, cArn := range lcOutput.ClusterArns {
		lciInput := &opsee_aws_ecs.ListContainerInstancesInput{
			Cluster: aws.String(cArn),
		}
		resp, err := c.Bezos.Get(
			ctx,
			&opsee.BezosRequest{
				User:   user,
				Region: region,
				VpcId:  vpc,
				Input:  &opsee.BezosRequest_Ecs_ListContainerInstancesInput{lciInput},
			},
		)
		if err != nil {
			return nil, err
		}

		// TODO(greg): support paging. god help us.
		lciOutput := resp.GetEcs_ListContainerInstancesOutput()
		if lciOutput == nil {
			return nil, fmt.Errorf("error decoding aws response")
		}

		if len(lciOutput.ContainerInstanceArns) == 0 {
			logger.Info("no container instances found")
			return svcs, nil
		}

		dciInput := &opsee_aws_ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(cArn),
			ContainerInstances: lciOutput.ContainerInstanceArns,
		}
		resp, err = c.Bezos.Get(
			ctx,
			&opsee.BezosRequest{
				User:   user,
				Region: region,
				VpcId:  vpc,
				Input:  &opsee.BezosRequest_Ecs_DescribeContainerInstancesInput{dciInput},
			},
		)
		if err != nil {
			return nil, err
		}

		// TODO(greg): support paging. god help us.
		dciOutput := resp.GetEcs_DescribeContainerInstancesOutput()
		if dciOutput == nil {
			return nil, fmt.Errorf("error decoding aws response")
		}

		if len(dciOutput.ContainerInstances) == 0 {
			logger.Info("no container instances found")
		}

		if len(dciOutput.ContainerInstances) > 0 {
			instanceId := dciOutput.ContainerInstances[0].Ec2InstanceId
			input := &opsee_aws_ec2.DescribeInstancesInput{
				InstanceIds: []string{aws.StringValue(instanceId)},
			}

			resp, err := c.Bezos.Get(
				ctx,
				&opsee.BezosRequest{
					User:   user,
					Region: region,
					VpcId:  vpc,
					Input:  &opsee.BezosRequest_Ec2_DescribeInstancesInput{input},
				},
			)
			if err != nil {
				return nil, err
			}

			output := resp.GetEc2_DescribeInstancesOutput()
			if output != nil {
				// we have a container instance in the vpc we're in... now we can get
				// services for this cluster.
				lsInput := &opsee_aws_ecs.ListServicesInput{
					Cluster: aws.String(cArn),
				}

				resp, err := c.Bezos.Get(
					ctx,
					&opsee.BezosRequest{
						User:   user,
						Region: region,
						VpcId:  vpc,
						Input:  &opsee.BezosRequest_Ecs_ListServicesInput{lsInput},
					},
				)
				if err != nil {
					return nil, err
				}

				lsOutput := resp.GetEcs_ListServicesOutput()
				if lsOutput != nil {
					dsInput := &opsee_aws_ecs.DescribeServicesInput{
						Cluster:  aws.String(cArn),
						Services: lsOutput.ServiceArns,
					}

					resp, err := c.Bezos.Get(
						ctx,
						&opsee.BezosRequest{
							User:   user,
							Region: region,
							VpcId:  vpc,
							Input:  &opsee.BezosRequest_Ecs_DescribeServicesInput{dsInput},
						},
					)
					if err != nil {
						return nil, err
					}

					dsOutput := resp.GetEcs_DescribeServicesOutput()
					if dsOutput != nil {
						if len(dsOutput.Services) == 0 {
							logger.Info("no services found")
						}

						for _, s := range dsOutput.Services {
							svcs = append(svcs, s)
						}
					}

				}
			}
		}
	}

	return svcs, nil
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
