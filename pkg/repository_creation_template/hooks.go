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

package repository_creation_template

import (
	"context"

	"github.com/aws-controllers-k8s/cloudwatch-controller/pkg/tags"
)

// getTags retrieves the tags for a given RepositoryCreationTemplate
func (rm *resourceManager) getTags(
	ctx context.Context,
	resourceARN string,
) map[string]*string {
	tagsManager := tags.NewTagsManager(rm.sdkapi)
	tags, err := tagsManager.GetTags(ctx, resourceARN)
	if err != nil {
		return nil
	}
	return tags
}

// syncTags synchronizes the tags for a RepositoryCreationTemplate
func (rm *resourceManager) syncTags(
	ctx context.Context,
	latest *resource,
	desired *resource,
) error {
	if latest.ko.Status.TemplateARN == nil {
		return nil
	}

	resourceARN := *latest.ko.Status.TemplateARN
	latestTags := map[string]*string{}
	if latest.ko.Spec.Tags != nil {
		latestTags = latest.ko.Spec.Tags
	}

	desiredTags := map[string]*string{}
	if desired.ko.Spec.Tags != nil {
		desiredTags = desired.ko.Spec.Tags
	}

	tagsManager := tags.NewTagsManager(rm.sdkapi)
	return tagsManager.SyncTags(
		ctx,
		resourceARN,
		desiredTags,
		latestTags,
	)
}
