"""Tests for kafka_cleanup_junk_topics.py.

Validates:
- JUNK_TOPIC_RE only matches the kafka-go test-suite naming pattern
- collect_junk_topics filters correctly
- delete_topics reports failures
"""

import re
import unittest

import kafka_cleanup_junk_topics as kc


class FakeFuture:
    def __init__(self, exc=None):
        self.exc = exc

    def result(self):
        if self.exc is not None:
            raise self.exc
        return None


class FakeAdmin:
    def __init__(self, topics):
        self._topics = topics
        self.deleted = []

    def list_topics(self, timeout=15):
        md = type("MD", (), {})()
        md.topics = {t: None for t in self._topics}
        return md

    def delete_topics(self, topics, operation_timeout=30):
        self.deleted.extend(topics)
        return {t: FakeFuture() for t in topics}


class TestJunkTopicPattern(unittest.TestCase):
    def test_matches_kafka_go_hex16(self):
        self.assertTrue(kc.JUNK_TOPIC_RE.match("kafka-go-72eca281a07f0d21"))
        self.assertTrue(kc.JUNK_TOPIC_RE.match("kafka-go-0000000000000000"))
        self.assertTrue(kc.JUNK_TOPIC_RE.match("kafka-go-ffffffffffffffff"))

    def test_rejects_business_topics_and_misnamed(self):
        for t in [
            "task-created", "user-created", "task-created-dlt",
            "kafka-go-group-abcdef1234567890",  # group-id pattern, not topic
            "kafka-go-transactional-id-abc",    # transactional-id pattern
            "kafka-go-xyz",                     # not hex
            "kafka-go-12345",                   # too short
            "__consumer_offsets",
        ]:
            self.assertIsNone(kc.JUNK_TOPIC_RE.match(t), f"should not match: {t}")


class TestCollectJunkTopics(unittest.TestCase):
    def test_filters_only_junk(self):
        topics = [
            "kafka-go-72eca281a07f0d21",
            "kafka-go-66211f5705f0abcd",
            "task-created",
            "task-deleted-dlt",
            "__consumer_offsets",
        ]
        admin = FakeAdmin(topics)
        self.assertEqual(kc.collect_junk_topics(admin), [
            "kafka-go-66211f5705f0abcd",
            "kafka-go-72eca281a07f0d21",
        ])


class TestDeleteTopics(unittest.TestCase):
    def test_reports_ok_and_failures(self):
        class FailingAdmin(FakeAdmin):
            def delete_topics(self, topics, operation_timeout=30):
                self.deleted.extend(topics)
                return {t: FakeFuture(Exception(f"boom {t}")) for t in topics}

        admin = FailingAdmin(["kafka-go-0000000000000000"])
        ok, failures = kc.delete_topics(admin, ["kafka-go-0000000000000000"])
        self.assertEqual(ok, 0)
        self.assertEqual(len(failures), 1)
        self.assertIn("kafka-go-0000000000000000", failures[0])


if __name__ == "__main__":
    unittest.main()
