// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Code generated by ack-generate. DO NOT EDIT.

package v1alpha1

import (
	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Hack to avoid import errors during build...
var (
	_ = &metav1.Time{}
	_ = &aws.JSONValue{}
	_ = ackv1alpha1.AWSAccountID("")
)

// Represents the history of a specific alarm.
type AlarmHistoryItem struct {
	AlarmName *string      `json:"alarmName,omitempty"`
	AlarmType *string      `json:"alarmType,omitempty"`
	Timestamp *metav1.Time `json:"timestamp,omitempty"`
}

// An anomaly detection model associated with a particular CloudWatch metric,
// statistic, or metric math expression. You can use the model to display a
// band of expected, normal values when the metric is graphed.
type AnomalyDetector struct {
	Dimensions []*Dimension `json:"dimensions,omitempty"`
	MetricName *string      `json:"metricName,omitempty"`
	Namespace  *string      `json:"namespace,omitempty"`
}

// The details about a composite alarm.
type CompositeAlarm struct {
	ActionsEnabled                     *bool        `json:"actionsEnabled,omitempty"`
	ActionsSuppressedBy                *string      `json:"actionsSuppressedBy,omitempty"`
	ActionsSuppressedReason            *string      `json:"actionsSuppressedReason,omitempty"`
	ActionsSuppressor                  *string      `json:"actionsSuppressor,omitempty"`
	ActionsSuppressorExtensionPeriod   *int64       `json:"actionsSuppressorExtensionPeriod,omitempty"`
	ActionsSuppressorWaitPeriod        *int64       `json:"actionsSuppressorWaitPeriod,omitempty"`
	AlarmActions                       []*string    `json:"alarmActions,omitempty"`
	AlarmARN                           *string      `json:"alarmARN,omitempty"`
	AlarmConfigurationUpdatedTimestamp *metav1.Time `json:"alarmConfigurationUpdatedTimestamp,omitempty"`
	AlarmDescription                   *string      `json:"alarmDescription,omitempty"`
	AlarmName                          *string      `json:"alarmName,omitempty"`
	AlarmRule                          *string      `json:"alarmRule,omitempty"`
	InsufficientDataActions            []*string    `json:"insufficientDataActions,omitempty"`
	OKActions                          []*string    `json:"oKActions,omitempty"`
	StateReason                        *string      `json:"stateReason,omitempty"`
	StateReasonData                    *string      `json:"stateReasonData,omitempty"`
	StateTransitionedTimestamp         *metav1.Time `json:"stateTransitionedTimestamp,omitempty"`
	StateUpdatedTimestamp              *metav1.Time `json:"stateUpdatedTimestamp,omitempty"`
	StateValue                         *string      `json:"stateValue,omitempty"`
}

// Encapsulates the statistical data that CloudWatch computes from metric data.
type Datapoint struct {
	Timestamp *metav1.Time `json:"timestamp,omitempty"`
	Unit      *string      `json:"unit,omitempty"`
}

// A dimension is a name/value pair that is part of the identity of a metric.
// Because dimensions are part of the unique identifier for a metric, whenever
// you add a unique name/value pair to one of your metrics, you are creating
// a new variation of that metric. For example, many Amazon EC2 metrics publish
// InstanceId as a dimension name, and the actual instance ID as the value for
// that dimension.
//
// You can assign up to 30 dimensions to a metric.
type Dimension struct {
	Name  *string `json:"name,omitempty"`
	Value *string `json:"value,omitempty"`
}

// Represents filters for a dimension.
type DimensionFilter struct {
	Name  *string `json:"name,omitempty"`
	Value *string `json:"value,omitempty"`
}

// One data point related to one contributor.
//
// For more information, see GetInsightRuleReport (https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_GetInsightRuleReport.html)
// and InsightRuleContributor (https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_InsightRuleContributor.html).
type InsightRuleContributorDatapoint struct {
	Timestamp *metav1.Time `json:"timestamp,omitempty"`
}

// One data point from the metric time series returned in a Contributor Insights
// rule report.
//
// For more information, see GetInsightRuleReport (https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_GetInsightRuleReport.html).
type InsightRuleMetricDatapoint struct {
	Timestamp *metav1.Time `json:"timestamp,omitempty"`
}

// Contains the information that's required to enable a managed Contributor
// Insights rule for an Amazon Web Services resource.
type ManagedRule struct {
	Tags []*Tag `json:"tags,omitempty"`
}

// Represents a specific metric.
type Metric struct {
	Dimensions []*Dimension `json:"dimensions,omitempty"`
	MetricName *string      `json:"metricName,omitempty"`
	Namespace  *string      `json:"namespace,omitempty"`
}

// The details about a metric alarm.
type MetricAlarm_SDK struct {
	ActionsEnabled                     *bool              `json:"actionsEnabled,omitempty"`
	AlarmActions                       []*string          `json:"alarmActions,omitempty"`
	AlarmARN                           *string            `json:"alarmARN,omitempty"`
	AlarmConfigurationUpdatedTimestamp *metav1.Time       `json:"alarmConfigurationUpdatedTimestamp,omitempty"`
	AlarmDescription                   *string            `json:"alarmDescription,omitempty"`
	AlarmName                          *string            `json:"alarmName,omitempty"`
	ComparisonOperator                 *string            `json:"comparisonOperator,omitempty"`
	DatapointsToAlarm                  *int64             `json:"datapointsToAlarm,omitempty"`
	Dimensions                         []*Dimension       `json:"dimensions,omitempty"`
	EvaluateLowSampleCountPercentile   *string            `json:"evaluateLowSampleCountPercentile,omitempty"`
	EvaluationPeriods                  *int64             `json:"evaluationPeriods,omitempty"`
	EvaluationState                    *string            `json:"evaluationState,omitempty"`
	ExtendedStatistic                  *string            `json:"extendedStatistic,omitempty"`
	InsufficientDataActions            []*string          `json:"insufficientDataActions,omitempty"`
	MetricName                         *string            `json:"metricName,omitempty"`
	Metrics                            []*MetricDataQuery `json:"metrics,omitempty"`
	Namespace                          *string            `json:"namespace,omitempty"`
	OKActions                          []*string          `json:"oKActions,omitempty"`
	Period                             *int64             `json:"period,omitempty"`
	StateReason                        *string            `json:"stateReason,omitempty"`
	StateReasonData                    *string            `json:"stateReasonData,omitempty"`
	StateTransitionedTimestamp         *metav1.Time       `json:"stateTransitionedTimestamp,omitempty"`
	StateUpdatedTimestamp              *metav1.Time       `json:"stateUpdatedTimestamp,omitempty"`
	StateValue                         *string            `json:"stateValue,omitempty"`
	Statistic                          *string            `json:"statistic,omitempty"`
	Threshold                          *float64           `json:"threshold,omitempty"`
	ThresholdMetricID                  *string            `json:"thresholdMetricID,omitempty"`
	TreatMissingData                   *string            `json:"treatMissingData,omitempty"`
	Unit                               *string            `json:"unit,omitempty"`
}

// This structure is used in both GetMetricData and PutMetricAlarm. The supported
// use of this structure is different for those two operations.
//
// When used in GetMetricData, it indicates the metric data to return, and whether
// this call is just retrieving a batch set of data for one metric, or is performing
// a Metrics Insights query or a math expression. A single GetMetricData call
// can include up to 500 MetricDataQuery structures.
//
// When used in PutMetricAlarm, it enables you to create an alarm based on a
// metric math expression. Each MetricDataQuery in the array specifies either
// a metric to retrieve, or a math expression to be performed on retrieved metrics.
// A single PutMetricAlarm call can include up to 20 MetricDataQuery structures
// in the array. The 20 structures can include as many as 10 structures that
// contain a MetricStat parameter to retrieve a metric, and as many as 10 structures
// that contain the Expression parameter to perform a math expression. Of those
// Expression structures, one must have true as the value for ReturnData. The
// result of this expression is the value the alarm watches.
//
// Any expression used in a PutMetricAlarm operation must return a single time
// series. For more information, see Metric Math Syntax and Functions (https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/using-metric-math.html#metric-math-syntax)
// in the Amazon CloudWatch User Guide.
//
// Some of the parameters of this structure also have different uses whether
// you are using this structure in a GetMetricData operation or a PutMetricAlarm
// operation. These differences are explained in the following parameter list.
type MetricDataQuery struct {
	AccountID  *string `json:"accountID,omitempty"`
	Expression *string `json:"expression,omitempty"`
	ID         *string `json:"id,omitempty"`
	Label      *string `json:"label,omitempty"`
	// This structure defines the metric to be returned, along with the statistics,
	// period, and units.
	MetricStat *MetricStat `json:"metricStat,omitempty"`
	Period     *int64      `json:"period,omitempty"`
	ReturnData *bool       `json:"returnData,omitempty"`
}

// A GetMetricData call returns an array of MetricDataResult structures. Each
// of these structures includes the data points for that metric, along with
// the timestamps of those data points and other identifying information.
type MetricDataResult struct {
	ID    *string `json:"id,omitempty"`
	Label *string `json:"label,omitempty"`
}

// Encapsulates the information sent to either create a metric or add new values
// to be aggregated into an existing metric.
type MetricDatum struct {
	Dimensions []*Dimension `json:"dimensions,omitempty"`
	MetricName *string      `json:"metricName,omitempty"`
	Timestamp  *metav1.Time `json:"timestamp,omitempty"`
	Unit       *string      `json:"unit,omitempty"`
}

// Indicates the CloudWatch math expression that provides the time series the
// anomaly detector uses as input. The designated math expression must return
// a single time series.
type MetricMathAnomalyDetector struct {
	MetricDataQueries []*MetricDataQuery `json:"metricDataQueries,omitempty"`
}

// This structure defines the metric to be returned, along with the statistics,
// period, and units.
type MetricStat struct {
	// Represents a specific metric.
	Metric *Metric `json:"metric,omitempty"`
	Period *int64  `json:"period,omitempty"`
	Stat   *string `json:"stat,omitempty"`
	Unit   *string `json:"unit,omitempty"`
}

// This structure contains the configuration information about one metric stream.
type MetricStreamEntry struct {
	CreationDate   *metav1.Time `json:"creationDate,omitempty"`
	LastUpdateDate *metav1.Time `json:"lastUpdateDate,omitempty"`
}

// This structure contains a metric namespace and optionally, a list of metric
// names, to either include in a metric stream or exclude from a metric stream.
//
// A metric stream's filters can include up to 1000 total names. This limit
// applies to the sum of namespace names and metric names in the filters. For
// example, this could include 10 metric namespace filters with 99 metrics each,
// or 20 namespace filters with 49 metrics specified in each filter.
type MetricStreamFilter struct {
	Namespace *string `json:"namespace,omitempty"`
}

// This object contains the information for one metric that is to be streamed
// with additional statistics.
type MetricStreamStatisticsMetric struct {
	MetricName *string `json:"metricName,omitempty"`
	Namespace  *string `json:"namespace,omitempty"`
}

// Specifies one range of days or times to exclude from use for training an
// anomaly detection model.
type Range struct {
	EndTime   *metav1.Time `json:"endTime,omitempty"`
	StartTime *metav1.Time `json:"startTime,omitempty"`
}

// Designates the CloudWatch metric and statistic that provides the time series
// the anomaly detector uses as input.
type SingleMetricAnomalyDetector struct {
	Dimensions []*Dimension `json:"dimensions,omitempty"`
	MetricName *string      `json:"metricName,omitempty"`
	Namespace  *string      `json:"namespace,omitempty"`
}

// A key-value pair associated with a CloudWatch resource.
type Tag struct {
	Key   *string `json:"key,omitempty"`
	Value *string `json:"value,omitempty"`
}
