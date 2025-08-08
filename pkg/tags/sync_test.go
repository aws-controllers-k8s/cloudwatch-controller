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

package tags

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
)

type mockCloudWatchClient struct {
	cloudwatchiface.CloudWatchAPI
	mock.Mock
}

func (m *mockCloudWatchClient) ListTagsForResourceWithContext(ctx context.Context, input *cloudwatch.ListTagsForResourceInput, opts ...request.Option) (*cloudwatch.ListTagsForResourceOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*cloudwatch.ListTagsForResourceOutput), args.Error(1)
}

func (m *mockCloudWatchClient) TagResourceWithContext(ctx context.Context, input *cloudwatch.TagResourceInput, opts ...request.Option) (*cloudwatch.TagResourceOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*cloudwatch.TagResourceOutput), args.Error(1)
}

func (m *mockCloudWatchClient) UntagResourceWithContext(ctx context.Context, input *cloudwatch.UntagResourceInput, opts ...request.Option) (*cloudwatch.UntagResourceOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*cloudwatch.UntagResourceOutput), args.Error(1)
}

func TestGetTags(t *testing.T) {
	mockClient := new(mockCloudWatchClient)
	tm := NewTagsManager(mockClient)
	ctx := context.Background()
	resourceARN := "arn:aws:cloudwatch:us-west-2:123456789012:repository-creation-template/test-template"

	expectedTags := []*cloudwatch.Tag{
		{
			Key:   aws.String("key1"),
			Value: aws.String("value1"),
		},
		{
			Key:   aws.String("key2"),
			Value: aws.String("value2"),
		},
	}

	mockClient.On("ListTagsForResourceWithContext", ctx, &cloudwatch.ListTagsForResourceInput{
		ResourceARN: aws.String(resourceARN),
	}).Return(&cloudwatch.ListTagsForResourceOutput{
		Tags: expectedTags,
	}, nil)

	tags, err := tm.GetTags(ctx, resourceARN)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(tags))
	assert.Equal(t, "value1", *tags["key1"])
	assert.Equal(t, "value2", *tags["key2"])

	mockClient.AssertExpectations(t)
}

func TestSyncTags(t *testing.T) {
	mockClient := new(mockCloudWatchClient)
	tm := NewTagsManager(mockClient)
	ctx := context.Background()
	resourceARN := "arn:aws:cloudwatch:us-west-2:123456789012:repository-creation-template/test-template"

	// Test 1: Add new tags
	desiredTags := map[string]*string{
		"key1": aws.String("value1"),
		"key2": aws.String("value2"),
	}
	latestTags := map[string]*string{}

	mockClient.On("TagResourceWithContext", ctx, mock.MatchedBy(func(input *cloudwatch.TagResourceInput) bool {
		return *input.ResourceARN == resourceARN && len(input.Tags) == 2
	})).Return(&cloudwatch.TagResourceOutput{}, nil).Once()

	err := tm.SyncTags(ctx, resourceARN, desiredTags, latestTags)
	assert.NoError(t, err)

	// Test 2: Update existing tags and remove others
	desiredTags = map[string]*string{
		"key1": aws.String("newvalue1"),
	}
	latestTags = map[string]*string{
		"key1": aws.String("value1"),
		"key2": aws.String("value2"),
	}

	mockClient.On("TagResourceWithContext", ctx, mock.MatchedBy(func(input *cloudwatch.TagResourceInput) bool {
		return *input.ResourceARN == resourceARN && len(input.Tags) == 1 && *input.Tags[0].Key == "key1" && *input.Tags[0].Value == "newvalue1"
	})).Return(&cloudwatch.TagResourceOutput{}, nil).Once()

	mockClient.On("UntagResourceWithContext", ctx, mock.MatchedBy(func(input *cloudwatch.UntagResourceInput) bool {
		return *input.ResourceARN == resourceARN && len(input.TagKeys) == 1 && *input.TagKeys[0] == "key2"
	})).Return(&cloudwatch.UntagResourceOutput{}, nil).Once()

	err = tm.SyncTags(ctx, resourceARN, desiredTags, latestTags)
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
}

func TestConvertTagsToACKTags(t *testing.T) {
	tags := map[string]*string{
		"key1": aws.String("value1"),
		"key2": aws.String("value2"),
	}

	ackTags := ConvertTagsToACKTags(tags)

	assert.Equal(t, 2, len(ackTags))

	// Since map iteration order is not guaranteed, we need to check both keys exist
	foundKey1 := false
	foundKey2 := false

	for _, tag := range ackTags {
		if *tag.Key == "key1" {
			assert.Equal(t, "value1", *tag.Value)
			foundKey1 = true
		}
		if *tag.Key == "key2" {
			assert.Equal(t, "value2", *tag.Value)
			foundKey2 = true
		}
	}

	assert.True(t, foundKey1)
	assert.True(t, foundKey2)
}

func TestConvertACKTagsToTags(t *testing.T) {
	key1 := "key1"
	key2 := "key2"
	value1 := "value1"
	value2 := "value2"

	ackTags := []*ackv1alpha1.Tag{
		{
			Key:   &key1,
			Value: &value1,
		},
		{
			Key:   &key2,
			Value: &value2,
		},
	}

	tags := ConvertACKTagsToTags(ackTags)

	assert.Equal(t, 2, len(tags))
	assert.Equal(t, "value1", *tags["key1"])
	assert.Equal(t, "value2", *tags["key2"])
}
