package resolver

import (
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/opsee/basic/schema"
	opsee_aws_cloudwatch "github.com/opsee/basic/schema/aws/cloudwatch"
	opsee "github.com/opsee/basic/service"
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
	"golang.org/x/net/context"
)

type metricList []*schema.Metric

func (l metricList) Len() int           { return len(l) }
func (l metricList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l metricList) Less(i, j int) bool { return l[i].Timestamp.Millis() < l[j].Timestamp.Millis() }

func (c *Client) GetMetricStatistics(ctx context.Context, user *schema.User, region string, input *opsee_aws_cloudwatch.GetMetricStatisticsInput) (*schema.CloudWatchResponse, error) {
	resp, err := c.Bezos.Get(ctx, &opsee.BezosRequest{User: user, Region: region, VpcId: "global", Input: &opsee.BezosRequest_Cloudwatch_GetMetricStatisticsInput{input}})
	if err != nil {
		return nil, err
	}

	output := resp.GetCloudwatch_GetMetricStatisticsOutput()
	if output == nil {
		return nil, fmt.Errorf("error decoding aws response")
	}

	metrics := make([]*schema.Metric, len(output.Datapoints))
	for i, d := range output.Datapoints {
		var statistic string
		if len(input.Statistics) > 0 {
			statistic = input.Statistics[0]
		}

		metrics[i] = &schema.Metric{
			Name: aws.StringValue(input.MetricName),
			// we really need support for other things?
			Value:     aws.Float64Value(d.Average),
			Timestamp: d.Timestamp,
			Unit:      aws.StringValue(d.Unit),
			Statistic: statistic,
		}
	}

	sort.Sort(metricList(metrics))

	return &schema.CloudWatchResponse{
		Namespace: aws.StringValue(input.Namespace),
		Metrics:   metrics,
	}, nil
}

func (c *Client) GetCheckMetrics(ctx context.Context, user *schema.User, checkId, metricName string, ts0, ts1 *opsee_types.Timestamp, aggregator *opsee.Aggregator) ([]*schema.Metric, error) {
	req := &opsee.GetMetricsRequest{
		Metrics: []*opsee.QueryMetric{
			&opsee.QueryMetric{
				Name: metricName,
				GroupBy: []*opsee.GroupBy{
					&opsee.GroupBy{
						Name: "tag",
						Tags: []string{"region"},
					},
				},
				Tags: map[string]*opsee.StringList{
					"check": &opsee.StringList{Values: []string{checkId}},
				},
				Aggregators: []*opsee.Aggregator{
					aggregator,
				},
			},
		},
		CacheTime:     0,
		StartAbsolute: ts0,
		EndAbsolute:   ts1,
	}

	r, err := c.Marktricks.GetMetrics(ctx, req)
	if err != nil {
		return nil, err
	}

	// convert kairosdb types to schema.Metric
	var m []*schema.Metric
	for _, query := range r.Queries {
		for _, result := range query.Results {
			var tags []*schema.Tag
			for k, v := range result.Tags {
				if len(v.Values) > 0 {
					tags = append(tags, &schema.Tag{k, v.Values[0]})
				}
			}
			for _, dp := range result.Values {
				nm := &schema.Metric{
					Name:      result.Name,
					Value:     float64(dp.Value),
					Timestamp: dp.Timestamp,
					Unit:      "milliseconds",
					Tags:      tags,
				}
				m = append(m, nm)
			}
		}
	}
	return m, nil
}
