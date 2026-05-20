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

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackerrors "github.com/aws-controllers-k8s/runtime/pkg/errors"
	acktypes "github.com/aws-controllers-k8s/runtime/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"

	svcapitypes "github.com/aws-controllers-k8s/cloudwatch-controller/apis/v1alpha1"
)

var (
	_ = &ackerrors.MissingNameIdentifier
)

type resource struct {
	ko *svcapitypes.PrometheusRule
}

func (r *resource) Identifiers() acktypes.AWSResourceIdentifiers {
	return &resourceIdentifiers{r.ko.Status.ACKResourceMetadata}
}

func (r *resource) IsBeingDeleted() bool {
	return !r.ko.DeletionTimestamp.IsZero()
}

func (r *resource) RuntimeObject() rtclient.Object {
	return r.ko
}

func (r *resource) MetaObject() metav1.Object {
	return r.ko.GetObjectMeta()
}

func (r *resource) Conditions() []*ackv1alpha1.Condition {
	return r.ko.Status.Conditions
}

func (r *resource) ReplaceConditions(conditions []*ackv1alpha1.Condition) {
	r.ko.Status.Conditions = conditions
}

func (r *resource) SetObjectMeta(meta metav1.ObjectMeta) {
	r.ko.ObjectMeta = meta
}

func (r *resource) SetStatus(desired acktypes.AWSResource) {
	r.ko.Status = desired.(*resource).ko.Status
}

func (r *resource) SetIdentifiers(identifier *ackv1alpha1.AWSIdentifiers) error {
	if identifier.NameOrID == "" {
		return ackerrors.MissingNameIdentifier
	}
	r.ko.Name = identifier.NameOrID
	return nil
}

func (r *resource) PopulateResourceFromAnnotation(fields map[string]string) error {
	name, ok := fields["name"]
	if !ok {
		return fmt.Errorf("required field missing: name")
	}
	r.ko.Name = name
	return nil
}

func (r *resource) DeepCopy() acktypes.AWSResource {
	return &resource{ko: r.ko.DeepCopy()}
}
