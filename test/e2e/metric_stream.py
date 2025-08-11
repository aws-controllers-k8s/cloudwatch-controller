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

"""Utilities for working with Metric Stream resources"""

import datetime
import time

import boto3
import pytest

DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS = 60*20
DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS = 15


def wait_until_deleted(
        metric_stream_name: str,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a Metric Stream with a supplied name is no longer returned from
    the CloudWatch API.

    Usage:
        from e2e.metric_stream import wait_until_deleted

        wait_until_deleted(stream_name)

    Raises:
        pytest.fail upon timeout or if the Metric Stream goes to any other status
        other than 'deleting'
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while True:
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                "Timed out waiting for Metric Stream to be "
                "deleted in CloudWatch API"
            )
        time.sleep(interval_seconds)

        latest = get(metric_stream_name)
        if latest is None:
            break


def exists(metric_stream_name):
    """Returns True if the supplied Metric Stream exists, False otherwise.
    """
    return get(metric_stream_name) is not None


def get(metric_stream_name):
    """Returns a dict containing the Metric Stream record from the CloudWatch API.

    If no such Metric Stream exists, returns None.
    """
    c = boto3.client('cloudwatch')
    try:
        resp = c.get_metric_stream(Name=metric_stream_name)
        return resp
    except c.exceptions.ResourceNotFoundException:
        return None