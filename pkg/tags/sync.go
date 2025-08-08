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

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
)

// TagsManager provides methods for working with CloudWatch resource tags
type TagsManager struct {
	// CloudWatchAPI provides the API operations for making requests to
	// CloudWatch service
	CloudWatchAPI cloudwatchiface.CloudWatchAPI
}

// NewTagsManager returns a new TagsManager for a resource
func NewTagsManager(api cloudwatchiface.CloudWatchAPI) *TagsManager {
	return &TagsManager{
		CloudWatchAPI: api,
	}
}

// GetTags returns the tags for a resource
func (tm *TagsManager) GetTags(
	ctx context.Context,
	resourceARN string,
) (map[string]*string, error) {
	input := &cloudwatch.ListTagsForResourceInput{
		ResourceARN: aws.String(resourceARN),
	}

	resp, err := tm.CloudWatchAPI.ListTagsForResourceWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	tags := map[string]*string{}
	for _, tag := range resp.Tags {
		tags[*tag.Key] = tag.Value
	}

	return tags, nil
}

// SyncTags synchronizes the tags for a resource by adding new tags,
// updating existing tags, and removing deleted tags
func (tm *TagsManager) SyncTags(
	ctx context.Context,
	resourceARN string,
	desiredTags map[string]*string,
	latestTags map[string]*string,
) error {
	// If there are no desired tags and no latest tags, there's nothing to do
	if len(desiredTags) == 0 && len(latestTags) == 0 {
		return nil
	}

	// Determine which tags to add or update
	tagsToAddOrUpdate := map[string]*string{}
	for k, v := range desiredTags {
		if latestValue, exists := latestTags[k]; !exists || *v != *latestValue {
			tagsToAddOrUpdate[k] = v
		}
	}

	// Determine which tags to remove
	var tagsToRemove []string
	for k := range latestTags {
		if _, exists := desiredTags[k]; !exists {
			tagsToRemove = append(tagsToRemove, k)
		}
	}

	// Add or update tags if needed
	if len(tagsToAddOrUpdate) > 0 {
		tagList := []*cloudwatch.Tag{}
		for k, v := range tagsToAddOrUpdate {
			tagList = append(tagList, &cloudwatch.Tag{
				Key:   aws.String(k),
				Value: v,
			})
		}

		_, err := tm.CloudWatchAPI.TagResourceWithContext(
			ctx,
			&cloudwatch.TagResourceInput{
				ResourceARN: aws.String(resourceARN),
				Tags:        tagList,
			},
		)
		if err != nil {
			return err
		}
	}

	// Remove tags if needed
	if len(tagsToRemove) > 0 {
		_, err := tm.CloudWatchAPI.UntagResourceWithContext(
			ctx,
			&cloudwatch.UntagResourceInput{
				ResourceARN: aws.String(resourceARN),
				TagKeys:     aws.StringSlice(tagsToRemove),
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// ConvertTagsToACKTags converts the CloudWatch tags to ACK tags
func ConvertTagsToACKTags(tags map[string]*string) []*ackv1alpha1.Tag {
	if len(tags) == 0 {
		return nil
	}

	res := make([]*ackv1alpha1.Tag, 0, len(tags))
	for k, v := range tags {
		res = append(res, &ackv1alpha1.Tag{
			Key:   &k,
			Value: v,
		})
	}

	return res
}

// ConvertACKTagsToTags converts ACK tags to CloudWatch tags
func ConvertACKTagsToTags(ackTags []*ackv1alpha1.Tag) map[string]*string {
	if len(ackTags) == 0 {
		return nil
	}

	res := make(map[string]*string, len(ackTags))
	for _, tag := range ackTags {
		res[*tag.Key] = tag.Value
	}

	return res
}
