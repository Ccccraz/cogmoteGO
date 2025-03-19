import subprocess
import time
import pytest
import requests
import json


from sseclient import SSEClient
from http import HTTPStatus
from faker import Faker


import logging

BASE_URL = "http://localhost:9012"
TEST_STREAM = {"name": "pytest-stream"}


def mock_data():
    """mock data"""
    faker = Faker()
    data = {"name": faker.name(), "email": faker.email(), "address": faker.address()}

    return data


@pytest.fixture(scope="module")
def mock_historic_data():
    """historic data"""
    datas = [mock_data() for _ in range(10)]
    yield datas


@pytest.fixture(scope="session", autouse=True)
def backend_server():
    """start the go server"""
    proc = subprocess.Popen(
        ["go", "run", "../cogmoteGO.go"],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )

    # wait for the server to start
    max_wait = 10
    for _ in range(max_wait):
        try:
            requests.get(f"{BASE_URL}/health")
            break
        except requests.ConnectionError:
            time.sleep(1)
    else:
        pytest.fail(f"Backend server did not start within {format(max_wait)} seconds")

    yield

    proc.terminate()
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        proc.kill()


def test_stream_creation():
    """test stream creation"""

    # create success
    resp = requests.post(f"{BASE_URL}/create", json=TEST_STREAM)
    assert resp.status_code == HTTPStatus.CREATED

    # create conflict
    conflict_resp = requests.post(f"{BASE_URL}/create", json=TEST_STREAM)
    assert conflict_resp.status_code == HTTPStatus.CONFLICT


def test_default_stream():
    """test default stream"""
    default = "data"

    data = mock_data()

    resp = requests.post(f"{BASE_URL}/{default}", json=data)
    assert resp.status_code == HTTPStatus.OK

    client = SSEClient(f"{BASE_URL}/{default}")
    assert json.loads(next(client).data) == data


def test_data_posting(mock_historic_data):
    """test data posting"""
    for data in mock_historic_data:
        resp = requests.post(f"{BASE_URL}/{TEST_STREAM['name']}", json=data)
        assert resp.status_code == HTTPStatus.OK, "Failed to post data to stream"


def test_historic_data_delivery(mock_historic_data):
    """test historic data delivery"""

    client = SSEClient(f"{BASE_URL}/{TEST_STREAM['name']}")

    # get historic messages

    msgs = [json.loads(next(client).data) for _ in range(10)]

    for mock, msg in zip(mock_historic_data, msgs):
        assert mock == msg, "Historic message doesn't match"
    client.resp.close()


def test_realtime_update():
    """test realtime update"""
    client = SSEClient(f"{BASE_URL}/{TEST_STREAM['name']}")

    data = mock_data()

    requests.post(f"{BASE_URL}/{TEST_STREAM['name']}", json=data)

    msgs = [json.loads(next(client).data) for _ in range(11)]

    assert msgs[10] == data, "Realtime message not received"
    client.resp.close()


def test_concurrent_subscriptions():
    """test concurrent subscriptions"""

    test_name = {"name": "concurrent-subscriptions"}
    requests.post(f"{BASE_URL}/create", json=test_name)
    logging.info(f"{test_name}: Successfully created {test_name} endpoint")

    client1 = SSEClient(f"{BASE_URL}/{test_name['name']}")
    client2 = SSEClient(f"{BASE_URL}/{test_name['name']}")

    data = mock_data()
    requests.post(f"{BASE_URL}/{test_name['name']}", json=data)

    assert json.loads(next(client1).data) == data
    assert json.loads(next(client2).data) == data
    client1.resp.close()
    client2.resp.close()


def test_invalid_stream_handling():
    """test invalid stream handling"""
    test_name = "invalid-stream-handling"

    data = mock_data()

    post_resp = requests.post(f"{BASE_URL}/{test_name}", json=data)
    assert post_resp.status_code == HTTPStatus.NOT_FOUND
