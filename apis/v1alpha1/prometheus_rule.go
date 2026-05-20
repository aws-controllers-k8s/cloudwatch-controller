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

package v1alpha1

import (
	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrometheusRuleSpec defines the desired state of PrometheusRule.
// The spec is intentionally compatible with the Prometheus Operator's
// PrometheusRule (monitoring.coreos.com/v1) to enable migration by
// changing only apiVersion/kind.
type PrometheusRuleSpec struct {
	// Groups of alerting rules to convert into CloudWatch MetricAlarms.
	// Each group maps to a set of alarms sharing a common prefix.
	// Recording rules are ignored (not supported by CloudWatch).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Groups []RuleGroup `json:"groups"`

	// CloudWatch-specific configuration for alarm actions and naming.
	// +kubebuilder:validation:Optional
	CloudWatch *CloudWatchConfig `json:"cloudWatch,omitempty"`
}

// RuleGroup is a list of sequentially evaluated alerting rules.
// Compatible with the Prometheus Operator RuleGroup spec.
type RuleGroup struct {
	// Name of the rule group. Used as part of the CloudWatch alarm name prefix.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Interval between rule evaluations. Maps to the CloudWatch alarm Period.
	// Accepts Prometheus duration format (e.g. "30s", "1m", "5m").
	// Defaults to "1m" if not specified.
	// +kubebuilder:validation:Optional
	Interval *string `json:"interval,omitempty"`

	// List of alerting rules. Recording rules (rules with a "record" field)
	// are silently skipped — CloudWatch does not support recording rules.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Rules []Rule `json:"rules"`
}

// Rule describes a single alerting rule.
// Compatible with the Prometheus Operator Rule spec.
type Rule struct {
	// Name of the alert. Required for alerting rules.
	// Used as the CloudWatch alarm name (prefixed with group name).
	// +kubebuilder:validation:Optional
	Alert *string `json:"alert,omitempty"`

	// PromQL expression to evaluate. This becomes the CloudWatch alarm's
	// metric math expression, evaluated against the CloudWatch PromQL engine.
	// +kubebuilder:validation:Required
	Expr string `json:"expr"`

	// Duration for which the condition must be true before firing.
	// Maps to CloudWatch alarm DatapointsToAlarm / EvaluationPeriods.
	// Accepts Prometheus duration format (e.g. "5m", "10m", "1h").
	// +kubebuilder:validation:Optional
	For *string `json:"for,omitempty"`

	// Labels to attach to the alert. Mapped to CloudWatch alarm tags.
	// The "severity" label is used to set alarm priority if present.
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations for the alert. The "summary" annotation maps to the
	// CloudWatch alarm description. Other annotations are stored as tags.
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Record is the name of a recording rule. If set, this rule is
	// silently skipped — CloudWatch does not support recording rules.
	// +kubebuilder:validation:Optional
	Record *string `json:"record,omitempty"`
}

// CloudWatchConfig holds CloudWatch-specific settings that don't exist
// in the Prometheus Operator spec. These are optional — the CRD works
// without them, but alarms won't have notification actions.
type CloudWatchConfig struct {
	// Actions to trigger when the alarm transitions to ALARM state.
	// Each entry is an ARN — supports SNS topics, Lambda functions,
	// SSM Incident Manager response plans, and Auto Scaling policies.
	// +kubebuilder:validation:Optional
	AlarmActions []string `json:"alarmActions,omitempty"`

	// Actions to trigger when the alarm transitions to OK state.
	// +kubebuilder:validation:Optional
	OKActions []string `json:"okActions,omitempty"`

	// Actions to trigger when the alarm transitions to INSUFFICIENT_DATA state.
	// +kubebuilder:validation:Optional
	InsufficientDataActions []string `json:"insufficientDataActions,omitempty"`

	// Prefix for CloudWatch alarm names. Defaults to the K8s namespace.
	// Final alarm name: {prefix}-{group}-{alert}
	// +kubebuilder:validation:Optional
	AlarmNamePrefix *string `json:"alarmNamePrefix,omitempty"`

	// Resource tags to apply to each CloudWatch alarm.
	// These are AWS resource tags, separate from Prometheus labels.
	// +kubebuilder:validation:Optional
	Tags map[string]string `json:"tags,omitempty"`
}

// PrometheusRuleStatus defines the observed state of PrometheusRule.
type PrometheusRuleStatus struct {
	// All CRs managed by ACK have a common `Status.ACKResourceMetadata` member
	// that is used to contain resource sync state, account ownership,
	// constructed ARN for the resource
	// +kubebuilder:validation:Optional
	ACKResourceMetadata *ackv1alpha1.ResourceMetadata `json:"ackResourceMetadata"`
	// All CRs managed by ACK have a common `Status.Conditions` member that
	// contains a collection of `ackv1alpha1.Condition` objects that describe
	// the various terminal states of the CR and its backend AWS service API
	// resource
	// +kubebuilder:validation:Optional
	Conditions []*ackv1alpha1.Condition `json:"conditions"`

	// Total number of alerting rules found across all groups.
	// +kubebuilder:validation:Optional
	AlertingRuleCount *int64 `json:"alertingRuleCount,omitempty"`

	// Number of rules skipped (recording rules).
	// +kubebuilder:validation:Optional
	SkippedRuleCount *int64 `json:"skippedRuleCount,omitempty"`

	// Per-alarm sync status. Key is the alarm name, value is the status.
	// +kubebuilder:validation:Optional
	AlarmStatuses []AlarmStatus `json:"alarmStatuses,omitempty"`
}

// AlarmStatus tracks the sync state of an individual CloudWatch alarm
// created from a Prometheus alerting rule.
type AlarmStatus struct {
	// CloudWatch alarm name.
	AlarmName string `json:"alarmName"`

	// Source rule group and alert name.
	RuleGroup string `json:"ruleGroup"`
	AlertName string `json:"alertName"`

	// Current state: Synced, Error, Pending.
	State string `json:"state"`

	// Error message if the alarm failed to sync.
	// +kubebuilder:validation:Optional
	Error *string `json:"error,omitempty"`

	// ARN of the created CloudWatch alarm.
	// +kubebuilder:validation:Optional
	AlarmARN *string `json:"alarmARN,omitempty"`
}

// PrometheusRule is the Schema for the PrometheusRules API.
// It accepts Prometheus alerting rules and converts them into
// CloudWatch MetricAlarms using PromQL metric math expressions.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="RULES",type=integer,JSONPath=`.status.alertingRuleCount`
// +kubebuilder:printcolumn:name="SKIPPED",type=integer,JSONPath=`.status.skippedRuleCount`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`
type PrometheusRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PrometheusRuleSpec   `json:"spec,omitempty"`
	Status            PrometheusRuleStatus `json:"status,omitempty"`
}

// PrometheusRuleList contains a list of PrometheusRule
// +kubebuilder:object:root=true
type PrometheusRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PrometheusRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PrometheusRule{}, &PrometheusRuleList{})
}
