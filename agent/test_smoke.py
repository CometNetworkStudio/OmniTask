import sys
from pathlib import Path

import grpc

sys.path.insert(0, str(Path(__file__).resolve().parent / "gen"))
sys.path.insert(0, str(Path(__file__).resolve().parent))

from agent.v1 import agent_pb2, agent_pb2_grpc  # noqa: E402
from grpc_server import VERSION, create_server  # noqa: E402


def fake_run(action, params, credentials):
    assert action == "noop"
    assert credentials == {"user": "u"}
    yield {"percent": 50, "message": "half", "result": {"k": "v"}}


def main():
    module = type("M", (), {"run": staticmethod(fake_run)})
    server = create_server({"echo": module})
    port = server.add_insecure_port("127.0.0.1:0")
    server.start()
    try:
        with grpc.insecure_channel(f"127.0.0.1:{port}") as channel:
            stub = agent_pb2_grpc.AgentServiceStub(channel)

            health = stub.Health(agent_pb2.HealthRequest())
            assert health.version == VERSION
            assert "echo" in health.platforms

            events = list(
                stub.Execute(
                    agent_pb2.ExecuteRequest(
                        task_id="t1",
                        platform="echo",
                        action="noop",
                        credentials={"user": "u"},
                    )
                )
            )
            types = [e.type for e in events]
            assert types[0] == agent_pb2.EVENT_TYPE_STARTED
            assert types[-1] == agent_pb2.EVENT_TYPE_SUCCEEDED
            assert any(
                e.type == agent_pb2.EVENT_TYPE_PROGRESS and e.percent == 50 for e in events
            )

            failed = list(
                stub.Execute(
                    agent_pb2.ExecuteRequest(task_id="t2", platform="missing", action="x")
                )
            )
            assert failed[-1].type == agent_pb2.EVENT_TYPE_FAILED
    finally:
        server.stop(0)
    print("agent smoke test ok")


if __name__ == "__main__":
    main()
