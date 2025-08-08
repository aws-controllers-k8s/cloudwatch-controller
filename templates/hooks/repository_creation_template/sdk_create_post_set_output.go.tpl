	// If tags were specified, mark the resource as needing to be synced
	if ko.Spec.Tags != nil {
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
	}