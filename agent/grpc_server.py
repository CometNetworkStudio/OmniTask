import importlib
import sys
import time
from concurrent import futures
from pathlib import Path

import grpc

sys.path.insert(0, str(Path(__file__).resolve().parent / "gen"))
sys.path.insert(0, str(Path(__file__).resolve().parent))

from agent.v1 import agent_pb2, agent_pb2_grpc  # noqa: E402

VERSION = "0.1.0"
DEFAULT_PORT = 50051
DEFAULT_PLATFORMS = ("cidaren", "welearn")


def load_platforms(names=DEFAULT_PLATFORMS):
    """按平台名加载 scripts.<platform>.runner，缺失则跳过。"""
    registry = {}
    for name in names:
        try:
            registry[name] = importlib.import_module(f"scripts.{name}.runner")
        except ImportError:
            continue
    return registry


class AgentService(agent_pb2_grpc.AgentServiceServicer):
    def __init__(self, platforms=None):
        self.platforms = platforms if platforms is not None else load_platforms()

    def Health(self, request, context):
        return agent_pb2.HealthResponse(version=VERSION, platforms=sorted(self.platforms))

    def Execute(self, request, context):
        now = int(time.time())
        yield agent_pb2.ExecuteEvent(
            task_id=request.task_id,
            type=agent_pb2.EVENT_TYPE_STARTED,
            percent=0,
            message=f"start {request.platform}/{request.action}",
            timestamp=now,
        )
        try:
            runner = self.platforms.get(request.platform)
            if runner is None:
                raise ValueError(f"unknown platform: {request.platform}")
            for event in runner.run(request.action, dict(request.params), dict(request.credentials)):
                yield agent_pb2.ExecuteEvent(
                    task_id=request.task_id,
                    type=agent_pb2.EVENT_TYPE_PROGRESS,
                    percent=int(event.get("percent", 0)),
                    message=event.get("message", ""),
                    timestamp=int(time.time()),
                    result={k: str(v) for k, v in (event.get("result") or {}).items()},
                )
        except Exception as exc:
            yield agent_pb2.ExecuteEvent(
                task_id=request.task_id,
                type=agent_pb2.EVENT_TYPE_FAILED,
                percent=0,
                message=f"{type(exc).__name__}: {exc}",
                timestamp=int(time.time()),
            )
            return
        yield agent_pb2.ExecuteEvent(
            task_id=request.task_id,
            type=agent_pb2.EVENT_TYPE_SUCCEEDED,
            percent=100,
            message="done",
            timestamp=int(time.time()),
        )


def create_server(platforms=None, max_workers=10):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=max_workers))
    agent_pb2_grpc.add_AgentServiceServicer_to_server(AgentService(platforms), server)
    return server


def serve(port=DEFAULT_PORT):
    server = create_server()
    server.add_insecure_port(f"[::]:{port}")
    server.start()
    print(f"agent listening on :{port} platforms={sorted(AgentService().platforms)}")
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
