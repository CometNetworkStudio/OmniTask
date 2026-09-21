import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from omnitask_sdk import AdapterContext, action, registered_actions  # noqa: E402
from omnitask_sdk import bridge  # noqa: E402


class DecoratorTest(unittest.TestCase):
    def test_register_action_and_context(self):
        @action("demo")
        def demo(ctx):
            ctx.progress(50, "half")
            ctx.result["ok"] = "1"

        handlers = registered_actions()
        self.assertIn("demo", handlers)

        events = []
        ctx = AdapterContext({"a": "1"}, {"t": "x"}, events.append, action="demo")
        handlers["demo"](ctx)

        self.assertEqual(events[0]["percent"], 50)
        self.assertEqual(events[0]["message"], "half")
        self.assertEqual(ctx.result["ok"], "1")


class BridgeTest(unittest.TestCase):
    def test_expand_placeholders(self):
        self.assertEqual(bridge._expand("{cid}", {"cid": "X"}, {}, "t", "run"), "X")
        self.assertEqual(bridge._expand("{token}", {}, {"token": "T"}, "t", "run"), "T")
        self.assertEqual(bridge._expand("{action}", {}, {}, "t", "run"), "run")
        self.assertEqual(bridge._expand("--course", {}, {}, "t", "run"), "--course")

    def test_find_action(self):
        manifest = {"actions": [{"id": "a", "command": ["x"]}]}
        self.assertEqual(bridge._find_action(manifest, "a")["command"], ["x"])
        self.assertIsNone(bridge._find_action(manifest, "b"))

    def test_compile_fallback(self):
        self.assertIsNotNone(bridge._compile(None))
        self.assertIsNotNone(bridge._compile("("))


if __name__ == "__main__":
    unittest.main()
