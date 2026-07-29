	// DescribeAlarms does not return tags — fetch them separately.
	if ko.Status.ACKResourceMetadata != nil && ko.Status.ACKResourceMetadata.ARN != nil {
		tagsInput := &svcsdk.ListTagsForResourceInput{
			ResourceARN: aws.String(string(*ko.Status.ACKResourceMetadata.ARN)),
		}
		tagsResp, tagsErr := rm.sdkapi.ListTagsForResource(ctx, tagsInput)
		rm.metrics.RecordAPICall("READ_MANY", "ListTagsForResource", tagsErr)
		if tagsErr != nil {
			return nil, tagsErr
		}
		ko.Spec.Tags = nil
		for _, t := range tagsResp.Tags {
			tCopy := svcapitypes.Tag{Key: t.Key, Value: t.Value}
			ko.Spec.Tags = append(ko.Spec.Tags, &tCopy)
		}
	}
