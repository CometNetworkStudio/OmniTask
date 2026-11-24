import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from omnitask_sdk.context import AdapterContext  # noqa: E402


class ContextComponentTest(unittest.TestCase):
    def setUp(self):
        self.events = []
        self.ctx = AdapterContext({"count": "3"}, {"token": "x"}, self.events.append, action="run")

    def test_progress_of_computes_percent(self):
        self.ctx.progress_of(3, 12, "处理中", unit="节")
        self.assertEqual(self.events[-1]["percent"], 25)
        self.assertEqual(self.events[-1]["current"], 3)
        self.assertEqual(self.events[-1]["total"], 12)
        self.assertEqual(self.events[-1]["unit"], "节")

    def test_progress_clamps(self):
        self.ctx.progress(120, "越界")
        self.assertEqual(self.events[-1]["percent"], 100)

    def test_step_status(self):
        self.ctx.step("login", "登录", "succeeded")
        event = self.events[-1]
        self.assertEqual(event["event"], "step")
        self.assertEqual(event["step"], "login")
        self.assertEqual(event["status"], "succeeded")

    def test_log_level_fallback(self):
        self.ctx.log("x", level="nope")
        self.assertEqual(self.events[-1]["event"], "log")
        self.assertEqual(self.events[-1]["level"], "info")

    def test_metric(self):
        self.ctx.metric("correct", "正确率", 96, "%")
        event = self.events[-1]
        self.assertEqual(event["event"], "metric")
        self.assertEqual(event["key"], "correct")
        self.assertEqual(event["value"], "96")
        self.assertEqual(event["unit"], "%")

    def test_cancel_event(self):
        import threading

        event = threading.Event()
        ctx = AdapterContext({}, {}, self.events.append, cancel_event=event)
        self.assertFalse(ctx.canceled)
        event.set()
        self.assertTrue(ctx.canceled)
        with self.assertRaises(Exception):
            ctx.check_canceled()

    def test_progress_of_zero_total_is_ignored(self):
        self.ctx.progress_of(0, 0)
        self.assertEqual(self.events[-1]["event"], "log")
        self.assertEqual(self.events[-1]["level"], "warn")


if __name__ == "__main__":
    unittest.main()
