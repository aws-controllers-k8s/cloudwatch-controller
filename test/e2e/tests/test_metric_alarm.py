# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Integration tests for the CloudWatch API MetricAlarm resource
"""

import time

import pytest

from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from acktest import tags
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_cloudwatch_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import condition
from e2e import metric_alarm

RESOURCE_PLURAL = 'metricalarms'

CHECK_STATUS_WAIT_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 5

def _make_metric_alarm(name_prefix: str, resource_name: str,
                       additional_replacements: dict = None):
    """Creates a MetricAlarm from a resource file and deletes it afterwards.

    Yields the (reference, custom resource) pair.
    """
    metric_alarm_name = random_suffix_name(name_prefix, 24)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["METRIC_ALARM_NAME"] = metric_alarm_name
    if additional_replacements:
        replacements.update(additional_replacements)
    resource_data = load_cloudwatch_resource(
        resource_name,
        additional_replacements=replacements,
    )

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        metric_alarm_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    _, deleted = k8s.delete_custom_resource(
        ref,
        period_length=DELETE_WAIT_AFTER_SECONDS,
    )
    assert deleted

    metric_alarm.wait_until_deleted(metric_alarm_name)


@pytest.fixture
def _metric_alarm():
    yield from _make_metric_alarm("ack-test-metric-alarm", "metric_alarm")


@pytest.fixture
def _evaluation_window_metric_alarm():
    yield from _make_metric_alarm(
        "ack-test-eval-window", "metric_alarm_evaluation_window")


@pytest.fixture
def _promql_metric_alarm():
    yield from _make_metric_alarm(
        "ack-test-promql-alarm", "metric_alarm_promql")


@pytest.fixture
def _tagged_metric_alarm():
    yield from _make_metric_alarm(
        "ack-test-tagged-alarm", "metric_alarm_with_tags",
        additional_replacements={"TAG_VALUE": "test"})


@service_marker
@pytest.mark.canary
class TestMetricAlarm:
    def test_crud(self, _metric_alarm):
        (ref, cr) = _metric_alarm
        metric_alarm_name = ref.name

        time.sleep(CHECK_STATUS_WAIT_SECONDS)

        condition.assert_synced(ref)

        assert metric_alarm.exists(metric_alarm_name)
        assert k8s.get_resource_exists(ref)


@service_marker
class TestMetricAlarmEvaluationWindow:
    """Covers the evaluationWindow union on a metric-based alarm.

    Kept on its own resource rather than added to TestMetricAlarm so that a
    gap in regional availability for evaluation windows cannot break the
    canary coverage for basic MetricAlarm CRUD.
    """

    def test_crud(self, _evaluation_window_metric_alarm):
        (ref, cr) = _evaluation_window_metric_alarm
        metric_alarm_name = ref.name

        time.sleep(CHECK_STATUS_WAIT_SECONDS)

        condition.assert_synced(ref)

        assert metric_alarm.exists(metric_alarm_name)
        assert k8s.get_resource_exists(ref)

        # Verify via the CR
        cr = k8s.get_resource(ref)
        assert cr["spec"]["evaluationWindow"]["wallClockWindow"]["timezone"] == 'America/New_York', \
            f"Expected wallClockWindow timezone on CR, got {cr['spec'].get('evaluationWindow')}"

        # Verify via the AWS API
        initial_alarm = metric_alarm.get(metric_alarm_name)
        assert initial_alarm is not None, "MetricAlarm not found in AWS API"

        initial_window = initial_alarm.get('EvaluationWindow', {})
        assert 'WallClockWindow' in initial_window, \
            f"Expected WallClockWindow on create, got {initial_window}"
        assert initial_window['WallClockWindow'].get('Timezone') == 'America/New_York', \
            f"Expected America/New_York on create, got {initial_window['WallClockWindow']}"

        # Update the timezone. This stays within the wallClockWindow member -
        # a fixed UTC offset must be aligned to a multiple of 5 minutes.
        updates = {
            "spec": {
                "evaluationWindow": {
                    "wallClockWindow": {
                        "timezone": "UTC+05:30"
                    }
                }
            }
        }

        k8s.patch_custom_resource(ref, updates)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        condition.assert_synced(ref)

        updated_alarm = metric_alarm.get(metric_alarm_name)
        assert updated_alarm is not None, "MetricAlarm not found in AWS API after update"

        updated_window = updated_alarm.get('EvaluationWindow', {})
        assert 'WallClockWindow' in updated_window, \
            f"Expected WallClockWindow after update, got {updated_window}"
        assert updated_window['WallClockWindow'].get('Timezone') == 'UTC+05:30', \
            f"Expected UTC+05:30 after update, got {updated_window['WallClockWindow']}"


@service_marker
class TestMetricAlarmTags:
    """Covers tag reconciliation on a MetricAlarm.

    PutMetricAlarm ignores the Tags field when updating an existing alarm, so
    tag changes are synced via the dedicated TagResource/UntagResource APIs.
    Kept on its own resource so a tagging regression cannot break the canary
    coverage for basic MetricAlarm CRUD.
    """

    def test_tag_sync(self, _tagged_metric_alarm):
        (ref, cr) = _tagged_metric_alarm
        metric_alarm_name = ref.name

        time.sleep(CHECK_STATUS_WAIT_SECONDS)
        condition.assert_synced(ref)

        assert metric_alarm.exists(metric_alarm_name)

        # The ARN is required to list tags in the AWS API.
        cr = k8s.get_resource(ref)
        metric_alarm_arn = cr["status"]["ackResourceMetadata"]["arn"]

        # Verify the create-time tag via the CR. Spec tags use the lowercase
        # key/value member names.
        tags.assert_present(
            expected={"env": "test"},
            actual=cr["spec"].get("tags"),
            key_member_name="key",
            value_member_name="value",
        )

        # Verify via the AWS API. assert_present tolerates the default
        # (services.k8s.aws/*) tags the controller also applies.
        aws_tags = metric_alarm.get_tags(metric_alarm_arn)
        tags.assert_present(expected={"env": "test"}, actual=aws_tags)
        # The controller must also apply the ACK system tags on create.
        tags.assert_ack_system_tags(tags=aws_tags)

        # Add a tag and update the value of the existing tag. Exercises
        # TagResource (PutMetricAlarm would silently drop these).
        updates = {
            "spec": {
                "tags": [
                    {"key": "env", "value": "prod"},
                    {"key": "team", "value": "platform"},
                ]
            }
        }
        k8s.patch_custom_resource(ref, updates)
        k8s.wait_resource_consumed_by_controller(ref)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        condition.assert_synced(ref)

        aws_tags = metric_alarm.get_tags(metric_alarm_arn)
        tags.assert_present(
            expected={"env": "prod", "team": "platform"},
            actual=aws_tags,
        )
        # The ACK system tags must survive the TagResource call.
        tags.assert_ack_system_tags(tags=aws_tags)

        # Remove a tag. Exercises UntagResource.
        updates = {
            "spec": {
                "tags": [
                    {"key": "env", "value": "prod"},
                ]
            }
        }
        k8s.patch_custom_resource(ref, updates)
        k8s.wait_resource_consumed_by_controller(ref)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        condition.assert_synced(ref)

        # Ignoring the ACK system tags, env=prod must be the only tag left -
        # this confirms both that team was removed and env=prod was retained.
        aws_tags = metric_alarm.get_tags(metric_alarm_arn)
        tags.assert_equal_without_ack_tags(
            expected={"env": "prod"},
            actual=aws_tags,
        )
        # The ACK system tags must survive the UntagResource call - only the
        # user-managed "team" tag should have been removed.
        tags.assert_ack_system_tags(tags=aws_tags)


@service_marker
class TestMetricAlarmPromQL:
    """Covers alarms configured with evaluationCriteria/evaluationInterval.

    These fields are mutually exclusive with the metric-based fields, so a
    PromQL alarm cannot be produced by patching the alarm in TestMetricAlarm -
    it needs its own resource.
    """

    def test_crud(self, _promql_metric_alarm):
        (ref, cr) = _promql_metric_alarm
        metric_alarm_name = ref.name

        time.sleep(CHECK_STATUS_WAIT_SECONDS)

        condition.assert_synced(ref)

        assert metric_alarm.exists(metric_alarm_name)
        assert k8s.get_resource_exists(ref)

        initial_query = 'max by (k8s_namespace_name) (container_cpu_utilization) > 90'

        # Verify via the CR
        cr = k8s.get_resource(ref)
        assert cr["spec"]["evaluationInterval"] == 60, \
            f"Expected evaluationInterval 60 on CR, got {cr['spec'].get('evaluationInterval')}"
        cr_promql = cr["spec"]["evaluationCriteria"]["promQLCriteria"]
        assert cr_promql["query"] == initial_query, \
            f"Expected query on CR, got {cr_promql.get('query')}"

        # Verify via the AWS API
        initial_alarm = metric_alarm.get(metric_alarm_name)
        assert initial_alarm is not None, "MetricAlarm not found in AWS API"

        assert initial_alarm.get('EvaluationInterval') == 60, \
            f"Expected EvaluationInterval 60, got {initial_alarm.get('EvaluationInterval')}"

        initial_criteria = initial_alarm.get('EvaluationCriteria', {})
        assert 'PromQLCriteria' in initial_criteria, \
            f"Expected PromQLCriteria on create, got {initial_criteria}"

        initial_promql = initial_criteria['PromQLCriteria']
        assert initial_promql.get('Query') == initial_query, \
            f"Expected query on create, got {initial_promql.get('Query')}"
        assert initial_promql.get('PendingPeriod') == 120, \
            f"Expected PendingPeriod 120 on create, got {initial_promql.get('PendingPeriod')}"
        assert initial_promql.get('RecoveryPeriod') == 120, \
            f"Expected RecoveryPeriod 120 on create, got {initial_promql.get('RecoveryPeriod')}"

        # Update the PromQL criteria and the evaluation interval. These stay
        # within the PromQL alarm shape - switching an alarm between the
        # metric-based and PromQL forms is not a supported update.
        updated_query = 'max by (k8s_namespace_name) (container_cpu_utilization) > 80'
        updates = {
            "spec": {
                "evaluationInterval": 120,
                "evaluationCriteria": {
                    "promQLCriteria": {
                        "query": updated_query,
                        "pendingPeriod": 300,
                        "recoveryPeriod": 120,
                    }
                }
            }
        }

        k8s.patch_custom_resource(ref, updates)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        condition.assert_synced(ref)

        updated_alarm = metric_alarm.get(metric_alarm_name)
        assert updated_alarm is not None, "MetricAlarm not found in AWS API after update"

        assert updated_alarm.get('EvaluationInterval') == 120, \
            f"Expected EvaluationInterval 120 after update, got {updated_alarm.get('EvaluationInterval')}"

        updated_criteria = updated_alarm.get('EvaluationCriteria', {})
        assert 'PromQLCriteria' in updated_criteria, \
            f"Expected PromQLCriteria after update, got {updated_criteria}"

        updated_promql = updated_criteria['PromQLCriteria']
        assert updated_promql.get('Query') == updated_query, \
            f"Expected updated query, got {updated_promql.get('Query')}"
        assert updated_promql.get('PendingPeriod') == 300, \
            f"Expected PendingPeriod 300 after update, got {updated_promql.get('PendingPeriod')}"
