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
	"sync"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcfg "github.com/aws-controllers-k8s/runtime/pkg/config"
	ackmetrics "github.com/aws-controllers-k8s/runtime/pkg/metrics"
	acktypes "github.com/aws-controllers-k8s/runtime/pkg/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/go-logr/logr"

	svcresource "github.com/aws-controllers-k8s/cloudwatch-controller/pkg/resource"
)

type resourceManagerFactory struct {
	sync.RWMutex
	rmCache map[string]*resourceManager
}

func (f *resourceManagerFactory) ResourceDescriptor() acktypes.AWSResourceDescriptor {
	return &resourceDescriptor{}
}

func (f *resourceManagerFactory) ManagerFor(
	cfg ackcfg.Config,
	clientcfg aws.Config,
	log logr.Logger,
	metrics *ackmetrics.Metrics,
	rr acktypes.Reconciler,
	id ackv1alpha1.AWSAccountID,
	region ackv1alpha1.AWSRegion,
	roleARN ackv1alpha1.AWSResourceName,
) (acktypes.AWSResourceManager, error) {
	rmId := fmt.Sprintf("%s/%s/%s", id, region, roleARN)
	f.RLock()
	rm, found := f.rmCache[rmId]
	f.RUnlock()

	if found {
		return rm, nil
	}

	f.Lock()
	defer f.Unlock()

	rm, err := newResourceManager(cfg, clientcfg, log, metrics, rr, id, region)
	if err != nil {
		return nil, err
	}
	f.rmCache[rmId] = rm
	return rm, nil
}

func (f *resourceManagerFactory) IsAdoptable() bool {
	return false
}

func (f *resourceManagerFactory) RequeueOnSuccessSeconds() int {
	return 0
}

func newResourceManagerFactory() *resourceManagerFactory {
	return &resourceManagerFactory{
		rmCache: map[string]*resourceManager{},
	}
}

func init() {
	svcresource.RegisterManagerFactory(newResourceManagerFactory())
}
