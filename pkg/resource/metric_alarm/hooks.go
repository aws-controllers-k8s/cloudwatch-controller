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

package metric_alarm

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// syncTags reconciles the tags on a MetricAlarm by calling TagResource for tags
// to add or update, and UntagResource for tags to remove.
// PutMetricAlarm silently ignores the Tags field on updates to existing alarms,
// so tag changes must be handled via the dedicated tag APIs.
func (rm *resourceManager) syncTags(
	ctx context.Context,
	desired *resource,
	latest *resource,
) error {
	desiredTags := desired.ko.Spec.Tags
	latestTags := latest.ko.Spec.Tags

	// Build lookup maps
	latestMap := make(map[string]string, len(latestTags))
	for _, t := range latestTags {
		if t.Key != nil {
			latestMap[*t.Key] = aws.ToString(t.Value)
		}
	}
	desiredMap := make(map[string]string, len(desiredTags))
	for _, t := range desiredTags {
		if t.Key != nil {
			desiredMap[*t.Key] = aws.ToString(t.Value)
		}
	}

	// Determine additions/updates and removals
	var addTags []svcsdktypes.Tag
	for k, v := range desiredMap {
		if latestV, ok := latestMap[k]; !ok || latestV != v {
			addTags = append(addTags, svcsdktypes.Tag{Key: aws.String(k), Value: aws.String(v)})
		}
	}
	var removeTags []string
	for k := range latestMap {
		if _, ok := desiredMap[k]; !ok {
			removeTags = append(removeTags, k)
		}
	}

	if len(addTags) == 0 && len(removeTags) == 0 {
		return nil
	}

	// ARN comes from the latest (observed) resource since desired may not have status populated.
	if latest.ko.Status.ACKResourceMetadata == nil || latest.ko.Status.ACKResourceMetadata.ARN == nil {
		return fmt.Errorf("cannot sync tags: MetricAlarm ARN is not yet available in status")
	}
	arn := string(*latest.ko.Status.ACKResourceMetadata.ARN)

	if len(addTags) > 0 {
		_, err := rm.sdkapi.TagResource(ctx, &svcsdk.TagResourceInput{
			ResourceARN: &arn,
			Tags:        addTags,
		})
		rm.metrics.RecordAPICall("UPDATE", "TagResource", err)
		if err != nil {
			return err
		}
	}

	if len(removeTags) > 0 {
		_, err := rm.sdkapi.UntagResource(ctx, &svcsdk.UntagResourceInput{
			ResourceARN: &arn,
			TagKeys:     removeTags,
		})
		rm.metrics.RecordAPICall("UPDATE", "UntagResource", err)
		if err != nil {
			return err
		}
	}

	return nil
}
