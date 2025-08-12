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

"""Integration tests for the CloudWatch API MetricStream resource
"""

import time

import pytest

from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_cloudwatch_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import condition
from e2e import metric_stream
from e2e.bootstrap_resources import get_bootstrap_resources

RESOURCE_PLURAL = 'metricstreams'

CHECK_STATUS_WAIT_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 30
DELETE_WAIT_AFTER_SECONDS = 5

@pytest.fixture
def _metric_stream():
    metric_stream_name = random_suffix_name("ack-test-metric-stream", 24)
    
    resources = get_bootstrap_resources()

    replacements = REPLACEMENT_VALUES.copy()
    replacements["METRIC_STREAM_NAME"] = metric_stream_name
    replacements["FIREHOSE_ARN"] = resources.DeliveryStream.arn
    replacements["ROLE_ARN"] = resources.MetricStreamRole.arn
    
    resource_data = load_cloudwatch_resource(
        "metric_stream",
        additional_replacements=replacements,
    )

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        metric_stream_name, namespace="default",
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

    metric_stream.wait_until_deleted(metric_stream_name)


@service_marker
@pytest.mark.canary
class TestMetricStream:
    def test_crud(self, _metric_stream):
        (ref, cr) = _metric_stream
        metric_stream_name = ref.name
        time.sleep(CHECK_STATUS_WAIT_SECONDS)
        condition.assert_synced(ref)

        assert metric_stream.exists(metric_stream_name)
        
        initial_stream_data = metric_stream.get(metric_stream_name)
        assert initial_stream_data is not None, "MetricStream not found in AWS API"
        initial_filters = initial_stream_data.get('IncludeFilters', [])
        assert len(initial_filters) == 2, f"Expected 2 initial filters, got {len(initial_filters)}: {initial_filters}"
        
        updates = {
            "spec": {
                "includeFilters": [
                    {"namespace": "AWS/EC2"}
                ]
            }
        }
        
        k8s.patch_custom_resource(ref, updates)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        condition.assert_synced(ref)

        assert metric_stream.exists(metric_stream_name)
        
        updated_stream_data = metric_stream.get(metric_stream_name)
        assert updated_stream_data is not None, "MetricStream not found in AWS API after update"
        
        updated_filters = updated_stream_data.get('IncludeFilters', [])
        assert len(updated_filters) == 1, f"Expected 1 filter after update, got {len(updated_filters)}: {updated_filters}"
        assert updated_filters[0]['Namespace'] == 'AWS/EC2', f"Expected AWS/EC2 filter, got {updated_filters[0]}"