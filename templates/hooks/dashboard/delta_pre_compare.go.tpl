    if ackcompare.HasNilDifference(a.ko.Spec.DashboardBody, b.ko.Spec.DashboardBody) {
		delta.Add("Spec.DashboardBody", a.ko.Spec.DashboardBody, b.ko.Spec.DashboardBody)
	} else if a.ko.Spec.DashboardBody != nil && b.ko.Spec.DashboardBody != nil {
		if !compareDashboardBody(*a.ko.Spec.DashboardBody, *b.ko.Spec.DashboardBody) {
			delta.Add("Spec.DashboardBody", a.ko.Spec.DashboardBody, b.ko.Spec.DashboardBody)
		}
	}