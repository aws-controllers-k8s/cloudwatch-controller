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

"""Integration tests for the CloudWatch API Dashboard resource
"""
import time
import json
import pytest

from acktest.k8s import resource as k8s, condition
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_cloudwatch_resource
from e2e.replacement_values import REPLACEMENT_VALUES

RESOURCE_PLURAL = 'dashboards'
CHECK_WAIT_SECONDS = 5

DASHBOARD_BODY_1 = r"""{
    "widgets": [
        {
            "type": "metric",
            "x": 0,
            "y": 0,
            "width": 12,
            "height": 6,
            "properties": {
            "metrics": [
                [ "AWS/EC2", "CPUUtilization", "InstanceId", "i-1234567890abcdef0" ]
            ],
            "period": 300,
            "stat": "Average",
            "region": "us-west-2",
            "title": "EC2"
            }
        }
    ]
}
"""

DASHBOARD_BODY_2 = r"""
{
    "widgets": [
        {
            "type": "metric",
            "x": 0,
            "y": 0,
            "width": 16,
            "height": 8,
            "properties": {
            "metrics": [
                [ "AWS/EKS", "scheduler_pending_pods_UNSCHEDULABLE", "ClusterName", "tutorial-cluster" ]
            ],
            "period": 150,
            "stat": "Count",
            "region": "us-west-2",
            "title": "EKS"
            }
        }
    ]
}
"""

def assert_json_equal(actual, expected):
    assert json.loads(actual) == json.loads(expected)

@pytest.fixture
def simple_dashboard():
    resource_name = random_suffix_name("ack-test-dashboard", 24)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["DASHBOARD_NAME"] = resource_name
    replacements["DASHBOARD_BODY"] = DASHBOARD_BODY_1

    resource_data = load_cloudwatch_resource(
        "dashboard",
        additional_replacements=replacements
    )

    print(resource_data)

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    _, deleted = k8s.delete_custom_resource(
        ref,
        wait_periods=5,
        period_length=5,
    )
    assert deleted

@service_marker
@pytest.mark.canary
class TestDashboard:
    def test_crud_dashboard(self, simple_dashboard, cloudwatch_client):
        (ref, cr) = simple_dashboard
        k8s.wait_on_condition(ref, condition.CONDITION_TYPE_RESOURCE_SYNCED, "True")
        condition.assert_synced(ref)

        cr = k8s.get_resource(ref)
        assert cr is not None
        assert 'dashboardBody' in cr['spec']
        assert_json_equal(cr['spec']['dashboardBody'], DASHBOARD_BODY_1)

        assert 'dashboardName' in cr['spec']
        dashboard_name = cr['spec']['dashboardName']

        new_dashboard = {
            "widgets": [
                {
                    "type": "metric",
                    "x": 0,
                    "y": 0,
                    "width": 16,
                    "height": 8,
                    "properties": {
                    "metrics": [
                        [ "AWS/EKS", "scheduler_pending_pods_UNSCHEDULABLE", "ClusterName", "test-cluster" ]
                    ],
                    "period": 120,
                    "stat": "Sum",
                    "region": "us-west-2",
                    "title": "EKS"
                    }
                }
            ]
        }

        dashboard = cloudwatch_client.get_dashboard(DashboardName=dashboard_name)
        assert dashboard is not None
        assert dashboard['DashboardName'] == dashboard_name
        assert_json_equal(dashboard['DashboardBody'], DASHBOARD_BODY_1)

        updates = {
            "spec": {
                "dashboardBody": json.dumps(new_dashboard),
            },
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(CHECK_WAIT_SECONDS)
        k8s.wait_on_condition(ref, condition.CONDITION_TYPE_RESOURCE_SYNCED, "True")
        cr = k8s.get_resource(ref)
        print(cr)
        condition.assert_synced(ref)

        cr = k8s.get_resource(ref)
        assert cr is not None
        assert 'dashboardBody' in cr['spec']
        assert_json_equal(cr['spec']['dashboardBody'], json.dumps(new_dashboard))

        dashboard = cloudwatch_client.get_dashboard(DashboardName=dashboard_name)
        assert dashboard is not None
        assert dashboard['DashboardName'] == dashboard_name
        assert_json_equal(dashboard['DashboardBody'], json.dumps(new_dashboard))
