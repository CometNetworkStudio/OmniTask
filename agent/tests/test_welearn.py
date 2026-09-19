import base64
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "gen"))
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from scripts.welearn.client import _extract  # noqa: E402
from scripts.welearn.crypto import generate_cipher_text  # noqa: E402
from scripts.welearn.runner import _collect_sco_ids, _is_actionable, _pick_course, _sco_id  # noqa: E402


class CryptoTest(unittest.TestCase):
    def test_golden_vectors(self):
        self.assertEqual(
            generate_cipher_text("abc", now_ms=1700000000123),
            ("MTcwMDAwMDAwMDEzMyo2MTYyNjM=", "1700000000133"),
        )
        self.assertEqual(
            generate_cipher_text("P@ssw0rd!", now_ms=1700000000123),
            ("MTcwMDAwMDAwMDEzMyo1MDQwNzM3Mzc3MzA3MjY0MjE=", "1700000000133"),
        )

    def test_structure(self):
        encoded, ts = generate_cipher_text("abc", now_ms=1700000000123)
        decoded = base64.b64decode(encoded).decode()
        prefix, hexpwd = decoded.split("*", 1)
        self.assertEqual(prefix, ts)
        self.assertEqual(hexpwd, "616263")

    def test_timestamp_millisecond_rounding(self):
        _, ts = generate_cipher_text("x", now_ms=1700000000999)
        self.assertTrue(ts.endswith(str(int(ts) % 100)))
        self.assertLessEqual(abs(int(ts) - 1700000000999), 99)


class ExtractTest(unittest.TestCase):
    def test_extract_encoded(self):
        url = "https://sso/a%26code_challenge%3DABC123%26state%3DXYZ%26b%3D1"
        self.assertEqual(_extract(url, "code_challenge"), "ABC123")
        self.assertEqual(_extract(url, "state"), "XYZ")

    def test_extract_missing(self):
        self.assertEqual(_extract("https://sso/a", "state"), "")


class RunnerHelpersTest(unittest.TestCase):
    def test_sco_id(self):
        self.assertEqual(_sco_id({"scoid": 123}), "123")
        self.assertEqual(_sco_id({"sco": "s1"}), "s1")
        self.assertIsNone(_sco_id({"name": "x"}))
        self.assertEqual(_sco_id("42"), "42")

    def test_pick_course(self):
        courses = [{"cid": "a"}, {"cid": "b"}]
        self.assertEqual(_pick_course(courses, None), {"cid": "a"})
        self.assertEqual(_pick_course(courses, "b"), {"cid": "b"})
        with self.assertRaises(Exception):
            _pick_course(courses, "z")
        with self.assertRaises(Exception):
            _pick_course([], None)

    def test_is_actionable(self):
        self.assertFalse(_is_actionable({"isvisible": "false", "iscomplete": "未完成"}, False))
        self.assertFalse(_is_actionable({"iscomplete": "已完成"}, False))
        self.assertTrue(_is_actionable({"iscomplete": "未完成"}, False))
        self.assertTrue(_is_actionable({"iscomplete": "已完成"}, True))
        self.assertTrue(_is_actionable("42", False))

    def test_collect_sco_ids_uses_numeric_index_and_dedupes(self):
        class FakeClient:
            def __init__(self, duplicate=False):
                self.calls = []
                self.duplicate = duplicate

            def get_sco_leaves(self, cid, uid, classid, unit_idx):
                self.calls.append(unit_idx)
                if self.duplicate:
                    return [{"id": "ITEM-dup", "isvisible": "true", "iscomplete": "未完成"}]
                return [
                    {"id": f"ITEM-{unit_idx}-0", "isvisible": "true", "iscomplete": "未完成"},
                    {"id": f"ITEM-{unit_idx}-1", "isvisible": "true", "iscomplete": "已完成"},
                ]

        units = [{"id": "GUID-A"}, {"id": "GUID-B"}]
        client = FakeClient()
        self.assertEqual(_collect_sco_ids(client, "c", "u", "cl", units, {}), ["ITEM-0-0", "ITEM-1-0"])
        self.assertEqual(client.calls, [0, 1])

        dup = FakeClient(duplicate=True)
        self.assertEqual(_collect_sco_ids(dup, "c", "u", "cl", units, {}), ["ITEM-dup"])

        included = FakeClient()
        ids = _collect_sco_ids(included, "c", "u", "cl", units, {"include_completed": "true"})
        self.assertEqual(ids, ["ITEM-0-0", "ITEM-0-1", "ITEM-1-0", "ITEM-1-1"])


if __name__ == "__main__":
    unittest.main()
