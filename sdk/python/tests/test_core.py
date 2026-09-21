import sys
import tempfile
import threading
import time
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from omnitask_sdk import events  # noqa: E402
from omnitask_sdk.bank import LocalBank  # noqa: E402
from omnitask_sdk.config import Config, load_dotenv  # noqa: E402
from omnitask_sdk.http import new_session  # noqa: E402
from omnitask_sdk.params import resolve  # noqa: E402
from omnitask_sdk.ratelimit import RateLimiter  # noqa: E402


class ParamsTest(unittest.TestCase):
    def test_defaults_and_overrides(self):
        p = resolve("cidaren", {"think_min": "3", "task_name": "List 04"})
        self.assertEqual(p["think_min"], 3.0)
        self.assertEqual(p["think_max"], 4.0)
        self.assertEqual(p["task_type"], 1)
        self.assertEqual(p["task_name"], "List 04")  # 未知参数透传

    def test_welearn_bool_and_int(self):
        p = resolve("welearn", {"duration": "10", "include_completed": "true"})
        self.assertEqual(p["duration"], 10)
        self.assertTrue(p["include_completed"])
        self.assertEqual(p["accuracy"], 100)

    def test_config_fallback_then_param_override(self):
        class FakeCfg:
            platform_concurrency = 5
            request_min_interval = 1.0
            request_max_interval = 2.0

        p = resolve("cidaren", {}, config=FakeCfg())
        self.assertEqual(p["platform_concurrency"], 5)
        p2 = resolve("cidaren", {"platform_concurrency": "1"}, config=FakeCfg())
        self.assertEqual(p2["platform_concurrency"], 1.0)


class EventsTest(unittest.TestCase):
    def test_event_clamps_percent(self):
        self.assertEqual(events.event(150, "x")["percent"], 100)
        self.assertEqual(events.event(-5, "x")["percent"], 0)
        self.assertEqual(events.event(50, "x")["result"], {})


class ConfigTest(unittest.TestCase):
    def test_load_dotenv_does_not_override(self):
        with tempfile.TemporaryDirectory() as tmp:
            env_file = Path(tmp) / ".env"
            env_file.write_text('# c\nA=1\nB="two"\nC=\n', encoding="utf-8")
            env = {"A": "keep"}
            load_dotenv(env_file, env=env)
            self.assertEqual(env["A"], "keep")
            self.assertEqual(env["B"], "two")
            self.assertEqual(env["C"], "")

    def test_from_env_uses_dict(self):
        env = {"OMNITASK_AGENT_PORT": "1234", "OMNITASK_PLATFORM_CONCURRENCY": "5"}
        cfg = Config.from_env(env=env)
        self.assertEqual(cfg.agent_port, 1234)
        self.assertEqual(cfg.platform_concurrency, 5)
        self.assertFalse(cfg.llm.enabled)


class RateLimiterTest(unittest.TestCase):
    def test_platform_concurrency_cap(self):
        limiter = RateLimiter(concurrency=2, min_interval=0, max_interval=0)
        current = 0
        peak = 0
        lock = threading.Lock()

        def worker():
            nonlocal current, peak
            with limiter.platform_slot("p"):
                with lock:
                    current += 1
                    peak = max(peak, current)
                time.sleep(0.02)
                with lock:
                    current -= 1

        threads = [threading.Thread(target=worker) for _ in range(6)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()
        self.assertLessEqual(peak, 2)

    def test_same_account_serial(self):
        limiter = RateLimiter(concurrency=4, min_interval=0, max_interval=0)
        current = 0
        peak = 0
        lock = threading.Lock()

        def worker():
            nonlocal current, peak
            with limiter.account_slot("p", "acc"):
                with lock:
                    current += 1
                    peak = max(peak, current)
                time.sleep(0.02)
                with lock:
                    current -= 1

        threads = [threading.Thread(target=worker) for _ in range(4)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()
        self.assertEqual(peak, 1)


class LocalBankTest(unittest.TestCase):
    def test_save_lookup_and_overwrite(self):
        with tempfile.TemporaryDirectory() as tmp:
            bank = LocalBank(str(Path(tmp) / "bank.sqlite3"))
            q = {
                "platform": "cidaren",
                "question_key": "k1",
                "stem": "hello",
                "options": '["a","b"]',
                "answer": '{"choice":0}',
                "source": "api",
                "confidence": 1.0,
            }
            self.assertTrue(bank.save(q))
            self.assertFalse(bank.save(q))  # 已存在且不覆盖
            hit = bank.lookup("cidaren", "k1")
            self.assertEqual(hit["answer"], '{"choice":0}')
            self.assertEqual(hit["hit_count"], 1)
            q["answer"] = '{"choice":1}'
            self.assertTrue(bank.save(q, overwrite=True))
            self.assertEqual(bank.lookup("cidaren", "k1")["answer"], '{"choice":1}')
            self.assertIsNone(bank.lookup("cidaren", "missing"))
            bank.close()


class HttpSessionTest(unittest.TestCase):
    def test_session_headers_and_proxy(self):
        session = new_session(headers={"X-Test": "1"}, proxy="http://127.0.0.1:9")
        self.assertEqual(session.headers["X-Test"], "1")
        self.assertIn("http", session.proxies)
        self.assertIsNotNone(session.get_adapter("https://example.com"))


if __name__ == "__main__":
    unittest.main()
