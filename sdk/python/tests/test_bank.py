import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from omnitask_sdk.gen.agent.v1 import agent_pb2  # noqa: E402
from omnitask_sdk.bank import GrpcBank  # noqa: E402


class FakeStub:
    def __init__(self, found=True):
        self.found = found
        self.saved = None

    def Lookup(self, request, timeout=None):
        if not self.found:
            return agent_pb2.LookupResponse(found=False)
        return agent_pb2.LookupResponse(
            found=True,
            question=agent_pb2.Question(
                platform=request.platform,
                question_key=request.question_key,
                stem="brutal",
                options="[]",
                answer='{"mode": 11, "corrects": [0]}',
                source="api",
                confidence=1.0,
                hit_count=3,
            ),
        )

    def Save(self, request, timeout=None):
        self.saved = request
        return agent_pb2.SaveResponse(saved=True)


class GrpcBankTest(unittest.TestCase):
    def test_lookup_mapping(self):
        bank = GrpcBank("unused", stub=FakeStub(), pb2=agent_pb2)
        hit = bank.lookup("cidaren", "k1")
        self.assertEqual(hit["answer"], '{"mode": 11, "corrects": [0]}')
        self.assertEqual(hit["hit_count"], 3)
        self.assertEqual(hit["source"], "api")

    def test_lookup_miss(self):
        bank = GrpcBank("unused", stub=FakeStub(found=False), pb2=agent_pb2)
        self.assertIsNone(bank.lookup("cidaren", "missing"))

    def test_save_serializes_objects(self):
        stub = FakeStub()
        bank = GrpcBank("unused", stub=stub, pb2=agent_pb2)
        ok = bank.save(
            {
                "platform": "cidaren",
                "question_key": "k1",
                "options": ["a", "b"],
                "answer": {"mode": 42, "corrects": [1]},
                "source": "api",
                "confidence": 1.0,
            },
            overwrite=True,
        )
        self.assertTrue(ok)
        self.assertTrue(stub.saved.overwrite)
        self.assertEqual(stub.saved.question.options, '["a", "b"]')
        self.assertEqual(stub.saved.question.answer, '{"mode": 42, "corrects": [1]}')


if __name__ == "__main__":
    unittest.main()
