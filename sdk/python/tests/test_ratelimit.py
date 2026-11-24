import os
import sys
import tempfile
import time
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from omnitask_sdk.ratelimit import SharedPacer  # noqa: E402


class SharedPacerTest(unittest.TestCase):
    def test_cross_instance_pacing(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, "rate.db")
            first = SharedPacer(path)
            second = SharedPacer(path)
            started = time.monotonic()
            first.wait("demo", 0.2, 0.2)
            second.wait("demo", 0.2, 0.2)
            self.assertGreaterEqual(time.monotonic() - started, 0.18)
            first.close()
            second.close()


if __name__ == "__main__":
    unittest.main()
