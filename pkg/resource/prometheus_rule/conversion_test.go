package prometheus_rule

import (
	"testing"

	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	svcapitypes "github.com/aws-controllers-k8s/cloudwatch-controller/apis/v1alpha1"
)

func strp(s string) *string { return &s }

func TestConvert_AlertingRule(t *testing.T) {
	spec := &svcapitypes.PrometheusRuleSpec{
		Groups: []svcapitypes.RuleGroup{
			{
				Name:     "test-group",
				Interval: strp("1m"),
				Rules: []svcapitypes.Rule{
					{
						Alert: strp("HighLatency"),
						Expr:  `histogram_quantile(0.99, sum by (le) (rate(http_duration_bucket[5m]))) > 2`,
						For:   strp("5m"),
						Labels: map[string]string{
							"severity": "critical",
						},
						Annotations: map[string]string{
							"summary": "High p99 latency detected",
						},
					},
				},
			},
		},
		CloudWatch: &svcapitypes.CloudWatchConfig{
			AlarmActions:    []string{"arn:aws:sns:us-east-1:123456789012:alerts"},
			AlarmNamePrefix: strp("myapp"),
		},
	}

	result, err := Convert("my-rule", "default", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Alarms) != 1 {
		t.Fatalf("expected 1 alarm, got %d", len(result.Alarms))
	}
	if result.SkippedCount != 0 {
		t.Fatalf("expected 0 skipped, got %d", result.SkippedCount)
	}

	alarm := result.Alarms[0]
	input := alarm.Input

	// Alarm name
	want := "myapp-test-group-HighLatency"
	if *input.AlarmName != want {
		t.Errorf("alarm name = %q, want %q", *input.AlarmName, want)
	}

	// EvaluationCriteria with PromQLCriteria
	ec, ok := input.EvaluationCriteria.(*svcsdktypes.EvaluationCriteriaMemberPromQLCriteria)
	if !ok {
		t.Fatalf("EvaluationCriteria is not PromQLCriteria, got %T", input.EvaluationCriteria)
	}
	if *ec.Value.Query != spec.Groups[0].Rules[0].Expr {
		t.Errorf("PromQL query mismatch")
	}

	// for: 5m = 300s PendingPeriod
	if *ec.Value.PendingPeriod != 300 {
		t.Errorf("PendingPeriod = %d, want 300", *ec.Value.PendingPeriod)
	}
	if *ec.Value.RecoveryPeriod != 300 {
		t.Errorf("RecoveryPeriod = %d, want 300", *ec.Value.RecoveryPeriod)
	}

	// EvaluationInterval from group interval (1m = 60s)
	if *input.EvaluationInterval != 60 {
		t.Errorf("EvaluationInterval = %d, want 60", *input.EvaluationInterval)
	}

	// Should NOT have classic alarm fields
	if input.Threshold != nil {
		t.Errorf("Threshold should be nil for PromQL alarms")
	}
	if len(input.Metrics) != 0 {
		t.Errorf("Metrics should be empty for PromQL alarms")
	}

	// SNS action
	if len(input.AlarmActions) != 1 || input.AlarmActions[0] != "arn:aws:sns:us-east-1:123456789012:alerts" {
		t.Errorf("alarm actions = %v, want SNS topic", input.AlarmActions)
	}

	// Description from annotation
	if *input.AlarmDescription != "High p99 latency detected" {
		t.Errorf("description = %q, want annotation summary", *input.AlarmDescription)
	}
}

func TestConvert_SkipsRecordingRules(t *testing.T) {
	spec := &svcapitypes.PrometheusRuleSpec{
		Groups: []svcapitypes.RuleGroup{
			{
				Name: "mixed",
				Rules: []svcapitypes.Rule{
					{
						Alert: strp("RealAlert"),
						Expr:  "up == 0",
					},
					{
						Record: strp("job:http_requests:rate5m"),
						Expr:   `sum by (job) (rate(http_requests_total[5m]))`,
					},
				},
			},
		},
	}

	result, err := Convert("rule", "ns", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Alarms) != 1 {
		t.Errorf("expected 1 alarm, got %d", len(result.Alarms))
	}
	if result.SkippedCount != 1 {
		t.Errorf("expected 1 skipped, got %d", result.SkippedCount)
	}
}

func TestConvert_DefaultPrefix(t *testing.T) {
	spec := &svcapitypes.PrometheusRuleSpec{
		Groups: []svcapitypes.RuleGroup{
			{
				Name:  "g",
				Rules: []svcapitypes.Rule{{Alert: strp("A"), Expr: "up"}},
			},
		},
	}

	result, err := Convert("rule", "production", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *result.Alarms[0].Input.AlarmName != "production-g-A" {
		t.Errorf("alarm name = %q, want namespace-based prefix", *result.Alarms[0].Input.AlarmName)
	}
}

func TestParseDurationSeconds(t *testing.T) {
	tests := []struct {
		input    string
		expected int32
	}{
		{"30s", 30},
		{"1m", 60},
		{"5m", 300},
		{"1h", 3600},
		{"1d", 86400},
		{"", 99},
		{"invalid", 99},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			s := tt.input
			got := parseDurationSeconds(&s, 99)
			if got != tt.expected {
				t.Errorf("parseDurationSeconds(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}
