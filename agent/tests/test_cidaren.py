import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "gen"))
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from scripts.cidaren import jv as jvmod  # noqa: E402
from scripts.cidaren import answer as answer_mod  # noqa: E402
from scripts.cidaren.sign import build_sign, md5_hex  # noqa: E402


class SignTest(unittest.TestCase):
    def test_small_golden(self):
        self.assertEqual(
            build_sign([("a", "1"), ("b", "2")]),
            "cc37c3ec0cf50560bbbc91c2d1a9db64",
        )
        self.assertEqual(build_sign({"x": "9"}), "6239a930e731b5b02fbb726136e44fec")

    def test_order_matters(self):
        self.assertNotEqual(build_sign([("a", "1"), ("b", "2")]), build_sign([("b", "2"), ("a", "1")]))

    def test_reference_vector(self):
        topic_code = (
            "k1ODfJVnV46DfnrEZ5Vol2lbXlrGnaebppqepJjKhmJoko9yaW2Ok2SWXpGSYZONZVyWbGdnbGlvZWiWamhvZJGW"
            "Ym6RlWxgmZRyZm2XbV1rXm9papuZa2WeYWdrbGtvY2qabmNoaW1sbm2akGaR"
        )
        params = [
            ("it_font_size", "42"),
            ("it_img_w", "804"),
            ("opt_font_c", "#000000"),
            ("opt_font_size", "37"),
            ("opt_img_w", "684"),
            ("time_spent", "25000"),
            ("timestamp", "1710407845296"),
            ("topic_code", topic_code),
            ("version", "2.6.2.24031302"),
        ]
        self.assertEqual(build_sign(params), "8daeb911bf2b8dd39ae8efb17515a96b")


class JvTest(unittest.TestCase):
    def test_known(self):
        self.assertTrue(jvmod.is_known_jv(""))
        self.assertTrue(jvmod.is_known_jv("0"))
        self.assertTrue(jvmod.is_known_jv("2_1254"))
        self.assertTrue(jvmod.is_known_jv("3_1021"))
        self.assertFalse(jvmod.is_known_jv("2_9999"))

    def test_strip_index(self):
        self.assertEqual(jvmod.strip_noise("abcdef", [0, 2, 4]), "bdf")

    def test_strip_segments(self):
        rules = [{"s": 0, "n": 1}, {"s": 2, "n": 2}]
        self.assertEqual(jvmod.strip_noise("abcdefg", rules), "bcfg")

    def test_decode_plain(self):
        payload = {"data": "eyJhIjogMX0="}  # {"a": 1}
        self.assertEqual(jvmod.decode(payload, ""), {"a": 1})

    def test_unknown_jv_raises(self):
        with self.assertRaises(jvmod.UnknownJv):
            jvmod.decode({"data": "!!!not-valid!!!", "jv": "2_9999"}, "2_9999")

    def test_roundtrip_jv_two(self):
        import base64
        import json

        body = json.dumps({"words": ["x" * 300], "k": 1})
        encoded = base64.b64encode(body.encode()).decode()
        indices = jvmod.JV_TWO["2_9214"]
        self.assertGreater(len(encoded), max(indices))
        chars = list(encoded)
        for index in sorted(indices):
            chars.insert(index, "X")
        obfuscated = "".join(chars)
        self.assertEqual(jvmod.decode({"data": obfuscated, "jv": "2_9214"}, "2_9214"), json.loads(body))


class AnswerTest(unittest.TestCase):
    def test_option_mode_record_and_resolve(self):
        exam = {
            "topic_mode": 42,
            "stem": {"content": "He {} the ball.", "remark": "他击球。"},
            "options": [{"content": "hit", "answer_tag": 0}, {"content": "run", "answer_tag": 1}],
        }
        record = answer_mod.derive_record(exam, {"answer_corrects": [0], "answer_result": 1})
        self.assertEqual(record["source"], "api")
        self.assertEqual(answer_mod.resolve_from_record(exam, record["answer"]), 0)

    def test_option_mode_reordered(self):
        exam = {
            "topic_mode": 42,
            "stem": {"content": "He {} the ball.", "remark": "他击球。"},
            "options": [{"content": "run", "answer_tag": 7}, {"content": "hit", "answer_tag": 3}],
        }
        record = answer_mod.derive_record(
            {"topic_mode": 42, "stem": exam["stem"], "options": [{"content": "hit"}, {"content": "run"}]},
            {"answer_corrects": [0]},
        )
        self.assertEqual(answer_mod.resolve_from_record(exam, record["answer"]), 3)

    def test_spelling_mode(self):
        exam = {"topic_mode": 73, "stem": {"content": "{ze} {wa}", "remark": ""}, "options": []}
        record = answer_mod.derive_record(exam, {"answer_corrects": ["zero", "waste"]})
        self.assertEqual(answer_mod.resolve_from_record(exam, record["answer"]), '["zero", "waste"]')

    def test_question_key_stable(self):
        exam = {"topic_mode": 32, "stem": {"content": "a _ b", "remark": "甲乙"}}
        self.assertEqual(answer_mod.question_key(exam), answer_mod.question_key(dict(exam)))
        self.assertNotEqual(answer_mod.question_key(exam), answer_mod.question_key({"topic_mode": 73, "stem": exam["stem"]}))


if __name__ == "__main__":
    unittest.main()
