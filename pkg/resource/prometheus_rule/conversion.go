// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/

package prometheus_rule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	svcsdk "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	svcapitypes "github.com/aws-controllers-k8s/cloudwatch-controller/apis/v1alpha1"
)

const defaultEvalIntervalSeconds int32 = 60

// ConvertResult holds the output of converting a PrometheusRule CR.
type ConvertResult struct {
	Alarms       []AlarmInput
	SkippedCount int64
}

// AlarmInput pairs a PutMetricAlarm input with source metadata.
type AlarmInput struct {
	Input     *svcsdk.PutMetricAlarmInput
	GroupName string
	AlertName string
}

// Convert translates a PrometheusRule spec into CloudWatch PutMetricAlarm inputs
// using the PromQL EvaluationCriteria API.
func Convert(name, namespace string, spec *svcapitypes.PrometheusRuleSpec) (*ConvertResult, error) {
	result := &ConvertResult{}
	prefix := alarmPrefix(name, namespace, spec.CloudWatch)

	for _, group := range spec.Groups {
		evalInterval := parseDurationSeconds(group.Interval, defaultEvalIntervalSeconds)

		for _, rule := range group.Rules {
			if rule.Record != nil {
				result.SkippedCount++
				continue
			}
			if rule.Alert == nil || *rule.Alert == "" {
				continue
			}

			input := ruleToAlarm(prefix, group.Name, evalInterval, &rule, spec.CloudWatch)
			result.Alarms = append(result.Alarms, AlarmInput{
				Input:     input,
				GroupName: group.Name,
				AlertName: *rule.Alert,
			})
		}
	}
	return result, nil
}

// AlarmName builds the CloudWatch alarm name from components.
func AlarmName(prefix, group, alert string) string {
	return fmt.Sprintf("%s-%s-%s", prefix, group, alert)
}

func ruleToAlarm(prefix, groupName string, evalInterval int32, rule *svcapitypes.Rule, cw *svcapitypes.CloudWatchConfig) *svcsdk.PutMetricAlarmInput {
	alarmName := AlarmName(prefix, groupName, *rule.Alert)
	description := buildDescription(rule)

	pendingPeriod := parseDurationSeconds(rule.For, 0)

	input := &svcsdk.PutMetricAlarmInput{
		AlarmName:        &alarmName,
		AlarmDescription: &description,
		EvaluationCriteria: &svcsdktypes.EvaluationCriteriaMemberPromQLCriteria{
			Value: svcsdktypes.AlarmPromQLCriteria{
				Query:          &rule.Expr,
				PendingPeriod:  &pendingPeriod,
				RecoveryPeriod: &pendingPeriod,
			},
		},
		EvaluationInterval: &evalInterval,
		ActionsEnabled:     boolPtr(true),
	}

	if cw != nil {
		input.AlarmActions = cw.AlarmActions
		input.OKActions = cw.OKActions
		input.InsufficientDataActions = cw.InsufficientDataActions

		// Resource tags from cloudWatch.tags
		for k, v := range cw.Tags {
			input.Tags = append(input.Tags, svcsdktypes.Tag{
				Key:   strPtr(k),
				Value: strPtr(v),
			})
		}
	}

	return input
}

func alarmPrefix(name, namespace string, cw *svcapitypes.CloudWatchConfig) string {
	if cw != nil && cw.AlarmNamePrefix != nil && *cw.AlarmNamePrefix != "" {
		return *cw.AlarmNamePrefix
	}
	return namespace
}

func buildDescription(rule *svcapitypes.Rule) string {
	var parts []string
	if rule.Annotations != nil {
		if s, ok := rule.Annotations["summary"]; ok {
			parts = append(parts, s)
		}
		if d, ok := rule.Annotations["description"]; ok && d != "" {
			parts = append(parts, d)
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("Prometheus alert: %s | expr: %s", *rule.Alert, rule.Expr)
	}
	return strings.Join(parts, " | ")
}

var durationRe = regexp.MustCompile(`^(\d+)(s|m|h|d)$`)

func parseDurationSeconds(d *string, fallback int32) int32 {
	if d == nil || *d == "" {
		return fallback
	}
	m := durationRe.FindStringSubmatch(*d)
	if m == nil {
		return fallback
	}
	val, err := strconv.Atoi(m[1])
	if err != nil {
		return fallback
	}
	switch m[2] {
	case "s":
		return int32(val)
	case "m":
		return int32(val * 60)
	case "h":
		return int32(val * 3600)
	case "d":
		return int32(val * 86400)
	}
	return fallback
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
