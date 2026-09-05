"""Tests for kafka_recreate.py topic recreation retry logic.

Kafka topic deletion is asynchronous at the broker level: after
delete_topics the topic disappears from list_topics metadata, but a
same-name create issued before the internal deletion completes fails
with TOPIC_ALREADY_EXISTS. The recreate step must retry that case
instead of failing the whole clear-db run.
"""

import sys
import types
import unittest
from types import SimpleNamespace
from unittest import mock

import kafka_recreate as kr


class FakeFuture:
    def __init__(self, exc=None):
        self.exc = exc

    def result(self):
        if self.exc is not None:
            raise self.exc
        return None


class FakeAdmin:
    def __init__(self, fail_once_topic="task-created"):
        self.fail_once_topic = fail_once_topic
        self.failed_once = False
        self.create_calls = []

    def create_topics(self, topics):
        self.create_calls.append([t.topic for t in topics])
        fs = {}
        for t in topics:
            if not self.failed_once and t.topic == self.fail_once_topic:
                self.failed_once = True
                fs[t.topic] = FakeFuture(Exception(
                    f"KafkaError{{code=TOPIC_ALREADY_EXISTS,val=36,str=\"Topic '{t.topic}' already exists.\"}}"
                ))
            else:
                fs[t.topic] = FakeFuture()
        return fs


class AlwaysFailsAdmin(FakeAdmin):
    def __init__(self):
        super().__init__()

    def create_topics(self, topics):
        self.create_calls.append([t.topic for t in topics])
        return {t.topic: FakeFuture(Exception(f"create {t.topic} failed: boom")) for t in topics}


class FakeNewTopic:
    def __init__(self, name, **kwargs):
        self.topic = name
        self.kwargs = kwargs


class TestCreateTopicsWithRetry(unittest.TestCase):
    def setUp(self):
        # Stub confluent_kafka.admin so tests run without a broker.
        admin_mod = types.ModuleType("confluent_kafka.admin")
        admin_mod.NewTopic = FakeNewTopic
        self.admin_patcher = mock.patch.dict(sys.modules, {"confluent_kafka.admin": admin_mod})
        self.admin_patcher.start()
        self.addCleanup(self.admin_patcher.stop)
        self.sleep_patcher = mock.patch("kafka_recreate.time.sleep")
        self.sleep_patcher.start()
        self.addCleanup(self.sleep_patcher.stop)

    def test_already_exists_is_retried_until_success(self):
        admin = FakeAdmin()
        topics = [SimpleNamespace(topic="task-created"), SimpleNamespace(topic="task-deleted")]
        ok = kr._create_topics_with_retry(admin, topics, max_wait=30)
        self.assertTrue(ok, "TOPIC_ALREADY_EXISTS should be retried, not fail the run")
        self.assertGreaterEqual(len(admin.create_calls), 2, "second attempt should have been made")
        self.assertTrue(admin.failed_once)

    def test_unrelated_errors_fail_fast(self):
        admin = AlwaysFailsAdmin()
        topics = [SimpleNamespace(topic="task-created")]
        ok = kr._create_topics_with_retry(admin, topics, max_wait=30)
        self.assertFalse(ok)
        self.assertEqual(len(admin.create_calls), 1, "non-deletion errors must not retry")


class TestReportKafkaConnectors(unittest.TestCase):
    """OPT-20260810-002: when topic deletion times out, name the processes
    holding active broker connections (zombie consumers defer deletion)."""

    def test_ss_holders_are_reported(self):
        with mock.patch("shutil.which", return_value="/usr/bin/ss"), \
             mock.patch("subprocess.run", return_value=SimpleNamespace(
                 stdout="State Recv-Q Send-Q Local Address:Port Peer Address:Port Process\n"
                        "ESTAB 0      0      10.2.150.68:9092  127.0.0.1:53122  users:((\"task-events-x\",pid=1234))\n")) as run:
            with mock.patch("builtins.print") as pr:
                kr._report_kafka_connectors()
        args = " ".join(str(a) for a in pr.call_args_list)
        self.assertIn("kafka:9092", args)
        self.assertIn("1234", args)
        self.assertIn("may block topic deletion", args)
        # ss must be invoked in established-state filter form
        self.assertIn("-tnp", " ".join(str(a) for a in run.call_args.args[0]))

    def test_no_holders_is_reported_quietly(self):
        with mock.patch("shutil.which", return_value="/usr/bin/ss"), \
             mock.patch("subprocess.run", return_value=SimpleNamespace(
                 stdout="State Recv-Q Send-Q Local Address:Port Peer Address:Port Process\n")):
            with mock.patch("builtins.print") as pr:
                kr._report_kafka_connectors()
        args = " ".join(str(a) for a in pr.call_args.args)
        self.assertIn("no established connections", args)

    def test_lsof_fallback(self):
        with mock.patch("shutil.which", side_effect=lambda x: None if x == "ss" else "/usr/bin/lsof"), \
             mock.patch("subprocess.run", return_value=SimpleNamespace(
                 stdout="COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME\n"
                        "task-events-x 1234 ljy 9u IPv4 0x0 0t0 TCP 127.0.0.1:53122->127.0.0.1:9092\n")):
            with mock.patch("builtins.print") as pr:
                kr._report_kafka_connectors()
        args = " ".join(str(a) for c in pr.call_args_list for a in c.args)
        self.assertIn("open files on kafka:9092", args)
        self.assertIn("1234", args)

    def test_no_tool_available(self):
        with mock.patch("shutil.which", return_value=None):
            with mock.patch("builtins.print") as pr:
                kr._report_kafka_connectors()
        args = " ".join(str(a) for a in pr.call_args.args)
        self.assertIn("ss/lsof unavailable", args)


if __name__ == "__main__":
    unittest.main()
