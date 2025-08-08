	if ko.Status.TemplateARN != nil {
		ko.Spec.Tags = rm.getTags(ctx, *ko.Status.TemplateARN)
	}