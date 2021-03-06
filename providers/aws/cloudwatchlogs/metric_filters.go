package cloudwatchlogs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)


type MetricFilter struct {
	ID uint `gorm:"primarykey"`
	AccountID string
	Region string
	CreationTime *int64
	FilterName *string
	FilterPattern *string
	LogGroupName *string
	Transformations []*MetricFilterTransformations `gorm:"constraint:OnDelete:CASCADE;"`
}

func (MetricFilter) TableName() string {
	return "aws_cloudwatchlogs_metric_filters"
}

type MetricFilterTransformations struct {
	ID uint `gorm:"primarykey"`
	AccountID string `gorm:"-"`
	Region string `gorm:"-"`
	MetricFilterID uint `neo:"ignore"`
	DefaultValue *float64
	MetricName *string
	MetricNamespace *string
	MetricValue *string
}

func (MetricFilterTransformations) TableName() string {
	return "aws_cloudwatchlogs_metric_filter_transformations"
}

func (c *Client) transformMetricFilters(values []*cloudwatchlogs.MetricFilter) []*MetricFilter {
	var tValues []*MetricFilter
	for _, value := range values {
		tValue := MetricFilter {
			AccountID: c.accountID,
			Region: c.region,
			CreationTime: value.CreationTime,
			FilterName: value.FilterName,
			FilterPattern: value.FilterPattern,
			LogGroupName: value.LogGroupName,
			Transformations: c.transformMetricFilterMetricTransformations(value.MetricTransformations),
		}
		tValues = append(tValues, &tValue)
	}
	return tValues
}

func (c *Client) transformMetricFilterMetricTransformations(values []*cloudwatchlogs.MetricTransformation) []*MetricFilterTransformations {
	var tValues []*MetricFilterTransformations
	for _, value := range values {
		tValue := MetricFilterTransformations{
			AccountID: c.accountID,
			Region: c.region,
			DefaultValue: value.DefaultValue,
			MetricName: value.MetricName,
			MetricNamespace: value.MetricNamespace,
			MetricValue: value.MetricValue,
		}
		tValues = append(tValues, &tValue)
	}
	return tValues
}
type MetricFilterConfig struct {
	Filter string
}

var MetricFilterTables = []interface{} {
	&MetricFilter{},
	&MetricFilterTransformations{},
}

func (c *Client)metricFilters(gConfig interface{}) error {
	var config cloudwatchlogs.DescribeMetricFiltersInput
	err := mapstructure.Decode(gConfig, &config)
	if err != nil {
		return err
	}
	c.db.Where("region", c.region).Where("account_id", c.accountID).Delete(MetricFilterTables...)

	for {
		output, err := c.svc.DescribeMetricFilters(&config)
		if err != nil {
			return err
		}
		c.db.ChunkedCreate(c.transformMetricFilters(output.MetricFilters))
		c.log.Info("Fetched resources", zap.String("resource", "cloudwatchlogs.metric_filters"), zap.Int("count", len(output.MetricFilters)))
		if aws.StringValue(output.NextToken) == "" {
			break
		}
		config.NextToken = output.NextToken
	}

	return nil
}

