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
"""Bootstraps the resources required to run the CloudWatch integration tests.
"""
import logging
import json

from acktest.bootstrapping import Resources, BootstrapFailureException
from e2e import bootstrap_directory
from e2e.bootstrap_resources import BootstrapResources
from acktest.bootstrapping.iam import Role, UserPolicies
from acktest.bootstrapping.firehose import DeliveryStream

def service_bootstrap() -> Resources:
    logging.getLogger().setLevel(logging.INFO)
    metric_stream_policy_doc = {
        "Version": "2012-10-17",
        "Statement": [{
            "Effect": "Allow",
            "Action": ["firehose:PutRecord", "firehose:PutRecordBatch"],
            "Resource": "*"
        }]
    }

    resources = BootstrapResources(
        MetricStreamRole=Role(
            name_prefix="cloudwatch-metric-stream-role",
            principal_service="streams.metrics.cloudwatch.amazonaws.com",
            description="Role for CloudWatch Metric Stream",
            user_policies=UserPolicies(
                name_prefix="metric-stream-firehose-policy",
                policy_documents=[json.dumps(metric_stream_policy_doc)]
            )
        ),
        DeliveryStream=DeliveryStream(
            name_prefix="cloudwatch-metric-stream",
            s3_bucket_prefix="ack-test-cw-metrics"
        )
    )

    try:
        resources.bootstrap()
    except BootstrapFailureException as ex:
        exit(254)

    return resources

if __name__ == "__main__":
    config = service_bootstrap()
    # Write config to current directory by default
    config.serialize(bootstrap_directory)
