// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/

package prometheus_rule

import (
	"reflect"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	acktypes "github.com/aws-controllers-k8s/runtime/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"
	k8sctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	svcapitypes "github.com/aws-controllers-k8s/cloudwatch-controller/apis/v1alpha1"
)

const (
	FinalizerString = "finalizers.cloudwatch.services.k8s.aws/PrometheusRule"
)

var (
	GroupVersionResource = svcapitypes.GroupVersion.WithResource("prometheusrules")
	GroupKind            = metav1.GroupKind{
		Group: "cloudwatch.services.k8s.aws",
		Kind:  "PrometheusRule",
	}
)

type resourceDescriptor struct{}

func (d *resourceDescriptor) GroupVersionKind() schema.GroupVersionKind {
	return svcapitypes.GroupVersion.WithKind(GroupKind.Kind)
}

func (d *resourceDescriptor) EmptyRuntimeObject() rtclient.Object {
	return &svcapitypes.PrometheusRule{}
}

func (d *resourceDescriptor) ResourceFromRuntimeObject(obj rtclient.Object) acktypes.AWSResource {
	return &resource{ko: obj.(*svcapitypes.PrometheusRule)}
}

func (d *resourceDescriptor) Delta(a, b acktypes.AWSResource) *ackcompare.Delta {
	desired := a.(*resource)
	latest := b.(*resource)
	delta := ackcompare.NewDelta()
	if !reflect.DeepEqual(desired.ko.Spec, latest.ko.Spec) {
		delta.Add("Spec", desired.ko.Spec, latest.ko.Spec)
	}
	return delta
}

func (d *resourceDescriptor) IsManaged(res acktypes.AWSResource) bool {
	obj := res.RuntimeObject()
	if obj == nil {
		panic("nil RuntimeMetaObject in AWSResource")
	}
	for _, f := range obj.GetFinalizers() {
		if f == FinalizerString {
			return true
		}
	}
	return false
}

func (d *resourceDescriptor) MarkManaged(res acktypes.AWSResource) {
	obj := res.RuntimeObject()
	if obj == nil {
		panic("nil RuntimeMetaObject in AWSResource")
	}
	k8sctrlutil.AddFinalizer(obj, FinalizerString)
}

func (d *resourceDescriptor) MarkUnmanaged(res acktypes.AWSResource) {
	obj := res.RuntimeObject()
	if obj == nil {
		panic("nil RuntimeMetaObject in AWSResource")
	}
	k8sctrlutil.RemoveFinalizer(obj, FinalizerString)
}

func (d *resourceDescriptor) MarkAdopted(res acktypes.AWSResource) {
	obj := res.RuntimeObject()
	if obj == nil {
		panic("nil RuntimeObject in AWSResource")
	}
	curr := obj.GetAnnotations()
	if curr == nil {
		curr = make(map[string]string)
	}
	curr[ackv1alpha1.AnnotationAdopted] = "true"
	obj.SetAnnotations(curr)
}
