package resolver

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/opsee/basic/schema"
	// log "github.com/sirupsen/logrus"
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
	"golang.org/x/net/context"
)

func (c *Client) GetMetricStatistics(ctx context.Context, user *schema.User, region string, input *cloudwatch.GetMetricStatisticsInput) (*schema.CloudWatchResponse, error) {
	sess, err := c.awsSession(ctx, user, region)
	if err != nil {
		return nil, err
	}

	output, err := cloudwatch.New(sess).GetMetricStatistics(input)
	if err != nil {
		return nil, err
	}

	metrics := make([]*schema.Metric, len(output.Datapoints))
	for i, d := range output.Datapoints {
		timestamp := &opsee_types.Timestamp{}
		timestamp.Scan(aws.TimeValue(d.Timestamp))

		var statistic string
		if len(input.Statistics) > 0 {
			statistic = aws.StringValue(input.Statistics[0])
		}

		metrics[i] = &schema.Metric{
			Name: aws.StringValue(input.MetricName),
			// we really need support for other things?
			Value:     aws.Float64Value(d.Average),
			Timestamp: timestamp,
			Unit:      aws.StringValue(d.Unit),
			Statistic: statistic,
		}
	}

	return &schema.CloudWatchResponse{
		Namespace: aws.StringValue(input.Namespace),
		Metrics:   metrics,
	}, nil
}
