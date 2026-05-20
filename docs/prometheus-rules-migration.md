# PrometheusRule Migration to CloudWatch

## Status

**Prototype** — April 2026

## Problem

Customers running Prometheus on Kubernetes have existing alerting rules defined as `PrometheusRule` CRDs (from the Prometheus Operator / kube-prometheus-stack). When migrating their metrics pipeline to CloudWatch (OTLP + PromQL), they lose their alerting rules because there's no native way to bring Prometheus alerting rules into CloudWatch.

This creates migration friction: customers must manually rewrite each alert as a CloudWatch MetricAlarm, translating PromQL expressions, evaluation periods, and notification routing by hand.

## Context

CloudWatch introduced:
- **OTLP metrics ingestion** — high-cardinality metrics with up to 150 labels
- **PromQL query support** — same query language customers already use
- **Automatic AWS resource enrichment** — account, region, cluster ARN, resource tags

This means CloudWatch can now evaluate the same PromQL expressions that Prometheus alerting rules use. The missing piece is a Kubernetes-native way to declare those rules.

### Prior Art: AMP Migration Pattern

The [ACK controller for Amazon Managed Service for Prometheus](https://aws.amazon.com/blogs/mt/migrating-to-amazon-managed-service-for-prometheus-with-the-prometheus-operator/) solved a similar problem with:
- `RuleGroupsNamespace` CRD — equivalent to Prometheus Operator's `PrometheusRule`
- `AlertManagerDefinition` CRD — equivalent to `AlertmanagerConfig`

Customers swap `apiVersion`/`kind`, add a `workspaceID`, and `kubectl apply`. The rule YAML body is passed through as-is because AMP runs a Prometheus-compatible backend.

### How CloudWatch Differs

CloudWatch doesn't run a Prometheus backend — it's CloudWatch with PromQL query support. So we can't just pass through the rule YAML. Instead, we **translate** each alerting rule into a CloudWatch MetricAlarm that uses the PromQL expression as a metric math query.

## Solution

Add a `PrometheusRule` CRD to the CloudWatch ACK controller (`cloudwatch.services.k8s.aws/v1alpha1`) that:

1. Accepts the same `groups[].rules[]` format as the Prometheus Operator
2. Translates each alerting rule into a CloudWatch MetricAlarm
3. Silently skips recording rules (not supported by CloudWatch)
4. Adds a `cloudWatch` section for CloudWatch-specific config (SNS topics, alarm naming)

### Migration Experience

```yaml
# BEFORE (Prometheus Operator)
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: app-alerts
spec:
  groups:
  - name: availability
    rules:
    - alert: HighErrorRate
      expr: sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) > 0.05
      for: "5m"
      labels:
        severity: critical
      annotations:
        summary: "Error rate above 5%"

# AFTER (CloudWatch) — change apiVersion/kind, add cloudWatch section
apiVersion: cloudwatch.services.k8s.aws/v1alpha1
kind: PrometheusRule
metadata:
  name: app-alerts
spec:
  groups:
  - name: availability
    rules:
    - alert: HighErrorRate
      expr: sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) > 0.05
      for: "5m"
      labels:
        severity: critical
      annotations:
        summary: "Error rate above 5%"
  cloudWatch:
    alarmSNSTopicARN: "arn:aws:sns:us-west-2:123456789012:alerts"
```

## Conversion Mapping

| Prometheus Concept | CloudWatch Equivalent | Notes |
|---|---|---|
| `alert` name | Alarm name (`{prefix}-{group}-{alert}`) | Prefix defaults to K8s namespace |
| `expr` (PromQL) | MetricAlarm metric math expression | Evaluated against CloudWatch PromQL store |
| `for` duration | `EvaluationPeriods` × `Period` | e.g. `5m` with `1m` period = 5 eval periods |
| `labels` | Alarm tags | `severity` label maps to priority |
| `annotations.summary` | Alarm description | |
| `annotations.description` | Appended to alarm description | |
| `interval` (group) | Alarm `Period` | Parsed from Prometheus duration format |
| Recording rules (`record`) | **Skipped** | Not supported by CloudWatch |
| Alertmanager routing | SNS topic ARN in `cloudWatch` config | Simpler model; use SNS filter policies for routing |

## CRD Design

### Spec

The spec has two sections:
- `groups[]` — identical to Prometheus Operator format for compatibility
- `cloudWatch` — optional CloudWatch-specific configuration

```go
type PrometheusRuleSpec struct {
    Groups     []RuleGroup      `json:"groups"`
    CloudWatch *CloudWatchConfig `json:"cloudWatch,omitempty"`
}
```

The `cloudWatch` section includes:
- `alarmSNSTopicARN` — SNS topic for ALARM notifications
- `okSNSTopicARN` — SNS topic for OK notifications
- `insufficientDataSNSTopicARN` — SNS topic for INSUFFICIENT_DATA
- `alarmNamePrefix` — prefix for alarm names (defaults to K8s namespace)
- `treatMissingData` — how to handle missing data points

### Status

```go
type PrometheusRuleStatus struct {
    ACKResourceMetadata *ackv1alpha1.ResourceMetadata
    Conditions          []*ackv1alpha1.Condition
    AlertingRuleCount   *int64
    SkippedRuleCount    *int64
    AlarmStatuses       []AlarmStatus
}
```

Each alarm's sync state is tracked individually so customers can see which rules were successfully created and which failed.

## Limitations

1. **Recording rules are not supported.** CloudWatch has no recording rules API. Customers should keep recording rules in their Prometheus instance or replace them with direct PromQL queries in CloudWatch (often fast enough given CloudWatch's query engine).

2. **Alertmanager routing complexity is not replicated.** Prometheus Alertmanager supports inhibition rules, grouping, silences, and complex routing trees. CloudWatch uses SNS for notifications, which is simpler. Advanced routing can be achieved with SNS filter policies or Lambda subscribers.

3. **Template variables in annotations are not supported.** Prometheus allows `{{ $value }}` and `{{ $labels.foo }}` in annotation templates. CloudWatch alarm descriptions are static strings.

4. **Threshold semantics.** Prometheus alerts fire when the expression returns a non-empty result (truthy). We map this to `threshold > 0` with `GreaterThanThreshold` comparison. This works for most cases but may need refinement for expressions that return negative values.

## File Structure

```
apis/v1alpha1/
  prometheus_rule.go          # CRD types
  zz_generated.deepcopy.go    # DeepCopy methods (appended)
generator.yaml                # PrometheusRule as custom resource
pkg/resource/prometheus_rule/
  conversion.go               # Rule → Alarm conversion logic
  conversion_test.go          # Unit tests
  resource.go                 # AWSResource wrapper
  identifiers.go              # Resource identifier implementation
  descriptor.go               # Resource descriptor with Delta comparison
  manager.go                  # Full CRUD resource manager
  manager_factory.go          # Factory registered via init()
cmd/controller/main.go        # Blank import to register resource
helm/crds/
  cloudwatch.services.k8s.aws_prometheusrules.yaml  # CRD for Helm
config/crd/bases/             # CRD for kustomize
config/rbac/                  # RBAC rules
samples/
  prometheus-rule-basic.yaml       # Minimal example
  prometheus-rule-migration.yaml   # Full before/after migration example
test/e2e/
  resources/prometheus_rule.yaml   # E2E test resource
  prometheus_rule.py               # E2E test helpers
docs/
  prometheus-rules-migration.md    # This document
```

## Reconciliation Flow

The resource manager implements the full ACK lifecycle:

```
kubectl apply PrometheusRule
        │
        ▼
  ┌─────────────┐
  │  Convert()   │  Parse spec.groups[].rules[]
  │              │  Skip recording rules (count them)
  │              │  Build PutMetricAlarm inputs
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │   Create()   │  For each alarm: PutMetricAlarm
  │              │  Track per-alarm status (Synced/Error)
  │              │  Set synthetic ARN on resource
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │  ReadOne()   │  DescribeAlarms for all expected names
  │              │  Rebuild status.alarmStatuses
  │              │  Return NotFound if zero alarms exist
  └──────┬──────┘
         │
    spec changed?
    ┌────┴────┐
    │ yes     │ no → done
    ▼         │
  ┌─────────────┐
  │  Update()   │  Re-convert spec → new alarm set
  │              │  PutMetricAlarm for each (idempotent)
  │              │  Delete alarms removed from spec (GC)
  └─────────────┘

kubectl delete PrometheusRule
        │
        ▼
  ┌─────────────┐
  │  Delete()   │  Convert to get all alarm names
  │              │  DeleteAlarms in batches of 100
  │              │  Ignore ResourceNotFound errors
  └─────────────┘
```

## Error Handling

- **Per-alarm errors**: If one alarm fails to create/update, the others still proceed. The failed alarm is tracked in `status.alarmStatuses` with `state: Error` and the error message. The overall resource gets a `Recoverable` condition so the controller retries.

- **Conversion errors**: If the spec is malformed (e.g. invalid duration format), the resource gets a `Terminal` condition and won't retry until the spec is fixed.

- **Garbage collection**: When rules are removed from the spec, the Update reconciler deletes the corresponding alarms. GC errors are logged but don't fail the update.

- **Batch deletion**: Delete handles up to 100 alarms per API call (CloudWatch limit). `ResourceNotFound` errors during deletion are ignored (idempotent).

## Alarm Lifecycle

Each Prometheus alerting rule maps to exactly one CloudWatch MetricAlarm:

```
Alarm name: {prefix}-{group}-{alert}
  │
  ├── prefix: cloudWatch.alarmNamePrefix or K8s namespace
  ├── group:  spec.groups[].name
  └── alert:  spec.groups[].rules[].alert
```

The alarm is **owned** by the PrometheusRule CR. When the CR is deleted, all its alarms are deleted. When a rule is removed from the spec, its alarm is garbage-collected on the next reconciliation.

## Future Work

- **Full reconciler implementation** — wire the conversion logic into an ACK resource manager with Create/Update/Delete/ReadOne
- **Drift detection** — detect when alarms are modified outside the controller
- **Recording rules** — if CloudWatch adds a recording rules API
- **Bulk migration CLI** — tool that reads existing `monitoring.coreos.com/v1` PrometheusRules from a cluster and outputs `cloudwatch.services.k8s.aws/v1alpha1` equivalents
- **Helm chart integration** — include the CRD in the controller's Helm chart
