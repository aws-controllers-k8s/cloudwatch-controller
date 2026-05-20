// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/

package prometheus_rule

import (
	"context"
	"errors"
	"fmt"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackcfg "github.com/aws-controllers-k8s/runtime/pkg/config"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackmetrics "github.com/aws-controllers-k8s/runtime/pkg/metrics"
	acktypes "github.com/aws-controllers-k8s/runtime/pkg/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	smithy "github.com/aws/smithy-go"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"

	svcapitypes "github.com/aws-controllers-k8s/cloudwatch-controller/apis/v1alpha1"
)

// +kubebuilder:rbac:groups=cloudwatch.services.k8s.aws,resources=prometheusrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudwatch.services.k8s.aws,resources=prometheusrules/status,verbs=get;update;patch

type resourceManager struct {
	cfg          ackcfg.Config
	clientcfg    aws.Config
	log          logr.Logger
	metrics      *ackmetrics.Metrics
	rr           acktypes.Reconciler
	awsAccountID ackv1alpha1.AWSAccountID
	awsRegion    ackv1alpha1.AWSRegion
	sdkapi       *svcsdk.Client
}

func (rm *resourceManager) concreteResource(res acktypes.AWSResource) *resource {
	return res.(*resource)
}

// ReadOne describes all alarms owned by this PrometheusRule and rebuilds status.
func (rm *resourceManager) ReadOne(
	ctx context.Context,
	res acktypes.AWSResource,
) (acktypes.AWSResource, error) {
	r := rm.concreteResource(res)
	if r.ko == nil {
		panic("resource manager's ReadOne() received resource with nil CR object")
	}

	// Convert spec to get expected alarm names.
	result, err := Convert(r.ko.Name, r.ko.Namespace, &r.ko.Spec)
	if err != nil {
		return rm.onError(r, err)
	}
	if len(result.Alarms) == 0 {
		return rm.onError(r, ackerr.NotFound)
	}

	// Collect all expected alarm names.
	alarmNames := make([]string, 0, len(result.Alarms))
	for _, a := range result.Alarms {
		alarmNames = append(alarmNames, *a.Input.AlarmName)
	}

	// Describe them in CloudWatch.
	resp, err := rm.sdkapi.DescribeAlarms(ctx, &svcsdk.DescribeAlarmsInput{
		AlarmNames: alarmNames,
	})
	rm.metrics.RecordAPICall("READ_MANY", "DescribeAlarms", err)
	if err != nil {
		return rm.onError(r, err)
	}

	// Build a set of found alarms.
	found := make(map[string]svcsdktypes.MetricAlarm, len(resp.MetricAlarms))
	for _, ma := range resp.MetricAlarms {
		if ma.AlarmName != nil {
			found[*ma.AlarmName] = ma
		}
	}

	// If none of the expected alarms exist, the resource hasn't been created yet.
	if len(found) == 0 {
		return rm.onError(r, ackerr.NotFound)
	}

	// Build status from what we found.
	ko := r.ko.DeepCopy()
	rm.setStatusDefaults(ko)
	statuses := make([]svcapitypes.AlarmStatus, 0, len(result.Alarms))
	for _, a := range result.Alarms {
		name := *a.Input.AlarmName
		as := svcapitypes.AlarmStatus{
			AlarmName: name,
			RuleGroup: a.GroupName,
			AlertName: a.AlertName,
		}
		if ma, ok := found[name]; ok {
			as.State = "Synced"
			if ma.AlarmArn != nil {
				as.AlarmARN = ma.AlarmArn
			}
		} else {
			as.State = "Pending"
			errMsg := "alarm not found in CloudWatch"
			as.Error = &errMsg
		}
		statuses = append(statuses, as)
	}
	alertCount := int64(len(result.Alarms))
	ko.Status.AlertingRuleCount = &alertCount
	ko.Status.SkippedRuleCount = &result.SkippedCount
	ko.Status.AlarmStatuses = statuses

	return rm.onSuccess(&resource{ko: ko})
}

// Create converts all alerting rules and creates CloudWatch alarms.
func (rm *resourceManager) Create(
	ctx context.Context,
	res acktypes.AWSResource,
) (acktypes.AWSResource, error) {
	r := rm.concreteResource(res)
	if r.ko == nil {
		panic("resource manager's Create() received resource with nil CR object")
	}

	result, err := Convert(r.ko.Name, r.ko.Namespace, &r.ko.Spec)
	if err != nil {
		return rm.onError(r, err)
	}

	ko := r.ko.DeepCopy()
	rm.setStatusDefaults(ko)
	statuses := make([]svcapitypes.AlarmStatus, 0, len(result.Alarms))

	for _, a := range result.Alarms {
		_, putErr := rm.sdkapi.PutMetricAlarm(ctx, a.Input)
		rm.metrics.RecordAPICall("CREATE", "PutMetricAlarm", putErr)

		as := svcapitypes.AlarmStatus{
			AlarmName: *a.Input.AlarmName,
			RuleGroup: a.GroupName,
			AlertName: a.AlertName,
		}
		if putErr != nil {
			as.State = "Error"
			errMsg := putErr.Error()
			as.Error = &errMsg
			// Continue creating remaining alarms — don't fail the whole batch.
		} else {
			as.State = "Synced"
			arn := rm.arnFromAlarmName(*a.Input.AlarmName)
			as.AlarmARN = &arn
		}
		statuses = append(statuses, as)
	}

	alertCount := int64(len(result.Alarms))
	ko.Status.AlertingRuleCount = &alertCount
	ko.Status.SkippedRuleCount = &result.SkippedCount
	ko.Status.AlarmStatuses = statuses

	// Set the resource ARN to a synthetic value based on the CR name.
	arn := ackv1alpha1.AWSResourceName(
		fmt.Sprintf("arn:aws:cloudwatch:%s:%s:prometheus-rule/%s",
			rm.awsRegion, rm.awsAccountID, ko.Name))
	ko.Status.ACKResourceMetadata.ARN = &arn

	// Check if any alarm failed.
	for _, s := range statuses {
		if s.State == "Error" {
			return rm.onError(&resource{ko: ko},
				fmt.Errorf("one or more alarms failed to create, check status.alarmStatuses"))
		}
	}

	return rm.onSuccess(&resource{ko: ko})
}

// Update re-converts the spec and puts all alarms. Alarms that no longer
// exist in the spec are deleted (garbage collection).
func (rm *resourceManager) Update(
	ctx context.Context,
	resDesired acktypes.AWSResource,
	resLatest acktypes.AWSResource,
	delta *ackcompare.Delta,
) (acktypes.AWSResource, error) {
	desired := rm.concreteResource(resDesired)
	latest := rm.concreteResource(resLatest)
	if desired.ko == nil || latest.ko == nil {
		panic("resource manager's Update() received resource with nil CR object")
	}

	// Convert desired spec to get the new set of alarms.
	result, err := Convert(desired.ko.Name, desired.ko.Namespace, &desired.ko.Spec)
	if err != nil {
		return rm.onError(desired, err)
	}

	// Build set of desired alarm names.
	desiredNames := make(map[string]struct{}, len(result.Alarms))
	for _, a := range result.Alarms {
		desiredNames[*a.Input.AlarmName] = struct{}{}
	}

	// Find alarms from the latest status that are no longer desired → delete them.
	if latest.ko.Status.AlarmStatuses != nil {
		var toDelete []string
		for _, as := range latest.ko.Status.AlarmStatuses {
			if _, ok := desiredNames[as.AlarmName]; !ok {
				toDelete = append(toDelete, as.AlarmName)
			}
		}
		if len(toDelete) > 0 {
			_, delErr := rm.sdkapi.DeleteAlarms(ctx, &svcsdk.DeleteAlarmsInput{
				AlarmNames: toDelete,
			})
			rm.metrics.RecordAPICall("DELETE", "DeleteAlarms", delErr)
			// Log but don't fail the update for GC errors.
		}
	}

	// Put all desired alarms (PutMetricAlarm is idempotent — creates or updates).
	ko := desired.ko.DeepCopy()
	rm.setStatusDefaults(ko)
	statuses := make([]svcapitypes.AlarmStatus, 0, len(result.Alarms))

	for _, a := range result.Alarms {
		_, putErr := rm.sdkapi.PutMetricAlarm(ctx, a.Input)
		rm.metrics.RecordAPICall("UPDATE", "PutMetricAlarm", putErr)

		as := svcapitypes.AlarmStatus{
			AlarmName: *a.Input.AlarmName,
			RuleGroup: a.GroupName,
			AlertName: a.AlertName,
		}
		if putErr != nil {
			as.State = "Error"
			errMsg := putErr.Error()
			as.Error = &errMsg
		} else {
			as.State = "Synced"
			arn := rm.arnFromAlarmName(*a.Input.AlarmName)
			as.AlarmARN = &arn
		}
		statuses = append(statuses, as)
	}

	alertCount := int64(len(result.Alarms))
	ko.Status.AlertingRuleCount = &alertCount
	ko.Status.SkippedRuleCount = &result.SkippedCount
	ko.Status.AlarmStatuses = statuses

	for _, s := range statuses {
		if s.State == "Error" {
			return rm.onError(&resource{ko: ko},
				fmt.Errorf("one or more alarms failed to update, check status.alarmStatuses"))
		}
	}

	return rm.onSuccess(&resource{ko: ko})
}

// Delete removes all CloudWatch alarms owned by this PrometheusRule.
func (rm *resourceManager) Delete(
	ctx context.Context,
	res acktypes.AWSResource,
) (acktypes.AWSResource, error) {
	r := rm.concreteResource(res)
	if r.ko == nil {
		panic("resource manager's Delete() received resource with nil CR object")
	}

	// Convert to get all alarm names.
	result, err := Convert(r.ko.Name, r.ko.Namespace, &r.ko.Spec)
	if err != nil {
		return nil, err
	}

	if len(result.Alarms) == 0 {
		return nil, nil
	}

	alarmNames := make([]string, 0, len(result.Alarms))
	for _, a := range result.Alarms {
		alarmNames = append(alarmNames, *a.Input.AlarmName)
	}

	// CloudWatch DeleteAlarms accepts up to 100 names per call.
	for i := 0; i < len(alarmNames); i += 100 {
		end := i + 100
		if end > len(alarmNames) {
			end = len(alarmNames)
		}
		_, err := rm.sdkapi.DeleteAlarms(ctx, &svcsdk.DeleteAlarmsInput{
			AlarmNames: alarmNames[i:end],
		})
		rm.metrics.RecordAPICall("DELETE", "DeleteAlarms", err)
		if err != nil {
			var awsErr smithy.APIError
			if errors.As(err, &awsErr) && awsErr.ErrorCode() == "ResourceNotFound" {
				continue // Already deleted, that's fine.
			}
			return nil, err
		}
	}

	return nil, nil
}

func (rm *resourceManager) ARNFromName(name string) string {
	return fmt.Sprintf("arn:aws:cloudwatch:%s:%s:prometheus-rule/%s",
		rm.awsRegion, rm.awsAccountID, name)
}

func (rm *resourceManager) arnFromAlarmName(name string) string {
	return fmt.Sprintf("arn:aws:cloudwatch:%s:%s:alarm:%s",
		rm.awsRegion, rm.awsAccountID, name)
}

// LateInitialize is a no-op for PrometheusRule.
func (rm *resourceManager) LateInitialize(
	_ context.Context,
	latest acktypes.AWSResource,
) (acktypes.AWSResource, error) {
	return latest, nil
}

// IsSynced checks if all alarms are in Synced state.
func (rm *resourceManager) IsSynced(_ context.Context, res acktypes.AWSResource) (bool, error) {
	r := rm.concreteResource(res)
	if r.ko == nil {
		return false, nil
	}
	for _, as := range r.ko.Status.AlarmStatuses {
		if as.State != "Synced" {
			return false, nil
		}
	}
	return true, nil
}

// EnsureTags is a no-op — tags are managed per-alarm via the conversion logic.
func (rm *resourceManager) EnsureTags(
	_ context.Context,
	_ acktypes.AWSResource,
	_ acktypes.ServiceControllerMetadata,
) error {
	return nil
}

// FilterSystemTags is a no-op for PrometheusRule.
func (rm *resourceManager) FilterSystemTags(_ acktypes.AWSResource, _ []string) {}

func (rm *resourceManager) setStatusDefaults(ko *svcapitypes.PrometheusRule) {
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if ko.Status.ACKResourceMetadata.Region == nil {
		ko.Status.ACKResourceMetadata.Region = &rm.awsRegion
	}
	if ko.Status.ACKResourceMetadata.OwnerAccountID == nil {
		ko.Status.ACKResourceMetadata.OwnerAccountID = &rm.awsAccountID
	}
	if ko.Status.Conditions == nil {
		ko.Status.Conditions = []*ackv1alpha1.Condition{}
	}
}

func (rm *resourceManager) onError(r *resource, err error) (acktypes.AWSResource, error) {
	if r == nil {
		return nil, err
	}
	r1, updated := rm.updateConditions(r, false, err)
	if !updated {
		return r, err
	}
	for _, condition := range r1.Conditions() {
		if condition.Type == ackv1alpha1.ConditionTypeTerminal &&
			condition.Status == corev1.ConditionTrue {
			return r1, ackerr.Terminal
		}
	}
	return r1, err
}

func (rm *resourceManager) onSuccess(r *resource) (acktypes.AWSResource, error) {
	if r == nil {
		return nil, nil
	}
	r1, updated := rm.updateConditions(r, true, nil)
	if !updated {
		return r, nil
	}
	return r1, nil
}

func (rm *resourceManager) updateConditions(
	r *resource,
	onSuccess bool,
	err error,
) (*resource, bool) {
	ko := r.ko.DeepCopy()
	rm.setStatusDefaults(ko)

	var terminalCondition *ackv1alpha1.Condition
	var recoverableCondition *ackv1alpha1.Condition
	var syncCondition *ackv1alpha1.Condition
	for _, condition := range ko.Status.Conditions {
		if condition.Type == ackv1alpha1.ConditionTypeTerminal {
			terminalCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeRecoverable {
			recoverableCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeResourceSynced {
			syncCondition = condition
		}
	}

	var termError *ackerr.TerminalError
	if errors.As(err, &termError) {
		if terminalCondition == nil {
			terminalCondition = &ackv1alpha1.Condition{
				Type: ackv1alpha1.ConditionTypeTerminal,
			}
			ko.Status.Conditions = append(ko.Status.Conditions, terminalCondition)
		}
		terminalCondition.Status = corev1.ConditionTrue
		msg := err.Error()
		terminalCondition.Message = &msg
	} else if terminalCondition != nil {
		terminalCondition.Status = corev1.ConditionFalse
		terminalCondition.Message = nil
	}

	if err != nil && !errors.As(err, &termError) {
		if recoverableCondition == nil {
			recoverableCondition = &ackv1alpha1.Condition{
				Type: ackv1alpha1.ConditionTypeRecoverable,
			}
			ko.Status.Conditions = append(ko.Status.Conditions, recoverableCondition)
		}
		recoverableCondition.Status = corev1.ConditionTrue
		msg := err.Error()
		recoverableCondition.Message = &msg
	} else if recoverableCondition != nil {
		recoverableCondition.Status = corev1.ConditionFalse
		recoverableCondition.Message = nil
	}

	if onSuccess {
		if syncCondition == nil {
			syncCondition = &ackv1alpha1.Condition{
				Type: ackv1alpha1.ConditionTypeResourceSynced,
			}
			ko.Status.Conditions = append(ko.Status.Conditions, syncCondition)
		}
		syncCondition.Status = corev1.ConditionTrue
		ackcondition.SetSynced(&resource{ko: ko}, corev1.ConditionTrue, nil, nil)
	} else if syncCondition != nil {
		syncCondition.Status = corev1.ConditionFalse
	}

	return &resource{ko: ko}, true
}

func newResourceManager(
	cfg ackcfg.Config,
	clientcfg aws.Config,
	log logr.Logger,
	metrics *ackmetrics.Metrics,
	rr acktypes.Reconciler,
	id ackv1alpha1.AWSAccountID,
	region ackv1alpha1.AWSRegion,
) (*resourceManager, error) {
	return &resourceManager{
		cfg:          cfg,
		clientcfg:    clientcfg,
		log:          log,
		metrics:      metrics,
		rr:           rr,
		awsAccountID: id,
		awsRegion:    region,
		sdkapi:       svcsdk.NewFromConfig(clientcfg),
	}, nil
}

// ClearResolvedReferences is a no-op — PrometheusRule has no reference fields.
func (rm *resourceManager) ClearResolvedReferences(res acktypes.AWSResource) acktypes.AWSResource {
	return res.DeepCopy()
}

// ResolveReferences is a no-op — PrometheusRule has no reference fields.
func (rm *resourceManager) ResolveReferences(
	ctx context.Context,
	apiReader rtclient.Reader,
	res acktypes.AWSResource,
) (acktypes.AWSResource, bool, error) {
	return res, false, nil
}
