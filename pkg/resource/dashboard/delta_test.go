package dashboard

import (
	"testing"

	svcapitypes "github.com/aws-controllers-k8s/cloudwatch-controller/apis/v1alpha1"
	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestNewResourceDelta_DashboardBody(t *testing.T) {
	tests := []struct {
		name     string
		bodyA    *string
		bodyB    *string
		wantDiff bool
	}{
		{
			name:     "identical JSON",
			bodyA:    aws.String(`{"widgets":[{"type":"metric","properties":{"metrics":["AWS/EC2","CPUUtilization"]}}]}`),
			bodyB:    aws.String(`{"widgets":[{"type":"metric","properties":{"metrics":["AWS/EC2","CPUUtilization"]}}]}`),
			wantDiff: false,
		},
		{
			name:     "equivalent JSON with different whitespace",
			bodyA:    aws.String(`{"widgets": [{"type": "metric", "properties": {"metrics": ["AWS/EC2", "CPUUtilization"]}}]}`),
			bodyB:    aws.String(`{"widgets":[{"type":"metric","properties":{"metrics":["AWS/EC2","CPUUtilization"]}}]}`),
			wantDiff: false,
		},
		{
			name:     "equivalent JSON with different key ordering",
			bodyA:    aws.String(`{"widgets":[{"type":"metric","properties":{"metrics":["AWS/EC2","CPUUtilization"]}}]}`),
			bodyB:    aws.String(`{"widgets":[{"properties":{"metrics":["AWS/EC2","CPUUtilization"]},"type":"metric"}]}`),
			wantDiff: false,
		},
		{
			name:     "different JSON content",
			bodyA:    aws.String(`{"widgets":[{"type":"metric","properties":{"metrics":["AWS/EC2","CPUUtilization"]}}]}`),
			bodyB:    aws.String(`{"widgets":[{"type":"text","properties":{"markdown":"Hello World"}}]}`),
			wantDiff: true,
		},
		{
			name:     "one invalid JSON",
			bodyA:    aws.String(`NOT JSON`),
			bodyB:    aws.String(`{"widgets":[{"type":"metric","properties":{"metrics":["AWS/EC2","CPUUtilization"]}}]}`),
			wantDiff: true,
		},
		{
			name:     "both invalid JSON but equal strings",
			bodyA:    aws.String(`NOT JSON`),
			bodyB:    aws.String(`NOT JSON`),
			wantDiff: false,
		},
		{
			name:     "both invalid JSON and differing strings",
			bodyA:    aws.String(`NOT JSON`),
			bodyB:    aws.String(`BUT DIFFER`),
			wantDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceA := &resource{
				ko: &svcapitypes.Dashboard{
					Spec: svcapitypes.DashboardSpec{
						DashboardBody: tt.bodyA,
					},
				},
			}
			resourceB := &resource{
				ko: &svcapitypes.Dashboard{
					Spec: svcapitypes.DashboardSpec{
						DashboardBody: tt.bodyB,
					},
				},
			}

			delta := newResourceDelta(resourceA, resourceB)
			hasDiff := len(delta.Differences) > 0

			if hasDiff != tt.wantDiff {
				t.Errorf("newResourceDelta() hasDiff = %v, want %v", hasDiff, tt.wantDiff)
			}
		})
	}
}
