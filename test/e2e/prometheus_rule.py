# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	 http://aws.amazon.com/apache2.0/

"""Utilities for working with PrometheusRule resources"""

import datetime
import time
from typing import List, Optional

import boto3
import pytest

DEFAULT_WAIT_TIMEOUT_SECONDS = 60 * 10
DEFAULT_WAIT_INTERVAL_SECONDS = 15


def get_alarm(alarm_name: str) -> Optional[dict]:
    """Returns a dict containing the MetricAlarm record from CloudWatch API.
    Returns None if not found.
    """
    c = boto3.client("cloudwatch")
    resp = c.describe_alarms(AlarmNames=[alarm_name])
    if len(resp["MetricAlarms"]) == 1:
        return resp["MetricAlarms"][0]
    return None


def alarm_exists(alarm_name: str) -> bool:
    return get_alarm(alarm_name) is not None


def get_expected_alarm_names(prefix: str, group: str, alerts: List[str]) -> List[str]:
    """Build expected alarm names from prefix, group, and alert names."""
    return [f"{prefix}-{group}-{alert}" for alert in alerts]


def wait_until_alarms_exist(
    alarm_names: List[str],
    timeout_seconds: int = DEFAULT_WAIT_TIMEOUT_SECONDS,
    interval_seconds: int = DEFAULT_WAIT_INTERVAL_SECONDS,
) -> None:
    """Waits until all supplied alarm names exist in CloudWatch."""
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while True:
        if datetime.datetime.now() >= timeout:
            missing = [n for n in alarm_names if not alarm_exists(n)]
            pytest.fail(
                f"Timed out waiting for alarms to be created. "
                f"Missing: {missing}"
            )
        time.sleep(interval_seconds)

        if all(alarm_exists(n) for n in alarm_names):
            break


def wait_until_alarms_deleted(
    alarm_names: List[str],
    timeout_seconds: int = DEFAULT_WAIT_TIMEOUT_SECONDS,
    interval_seconds: int = DEFAULT_WAIT_INTERVAL_SECONDS,
) -> None:
    """Waits until none of the supplied alarm names exist in CloudWatch."""
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while True:
        if datetime.datetime.now() >= timeout:
            remaining = [n for n in alarm_names if alarm_exists(n)]
            pytest.fail(
                f"Timed out waiting for alarms to be deleted. "
                f"Remaining: {remaining}"
            )
        time.sleep(interval_seconds)

        if not any(alarm_exists(n) for n in alarm_names):
            break
