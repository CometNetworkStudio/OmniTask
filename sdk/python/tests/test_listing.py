import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from omnitask_sdk import listing, registered_listings  # noqa: E402
from omnitask_sdk import runtime  # noqa: E402


class ListingTest(unittest.TestCase):
    def test_register(self):
        @listing("unit-test")
        def _f(ctx):
            return []

        self.assertIn("unit-test", registered_listings())

    def test_normalize_flat(self):
        item = runtime._normalize_item({"id": "a", "title": "T", "progress": 49})
        self.assertEqual(item["key"], "a")
        self.assertEqual(item["fields"]["progress"], "49")

    def test_normalize_explicit(self):
        item = runtime._normalize_item({"key": "k", "fields": {"a": 1}})
        self.assertEqual(item["key"], "k")
        self.assertEqual(item["fields"]["a"], "1")

    def test_normalize_scalar(self):
        self.assertEqual(runtime._normalize_item(7)["key"], "7")


if __name__ == "__main__":
    unittest.main()
