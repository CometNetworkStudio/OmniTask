from collections.abc import Iterator


def run(action: str, params: dict[str, str], credentials: dict[str, str]) -> Iterator[dict]:
    """平台脚本统一入口（净室实现）。

    - action: 平台内动作，如 "study_task"、"test_task"
    - params: 任务参数
    - credentials: 后端解密后下发的凭据，不得持久化

    以生成器方式 yield 进度事件，每个事件形如：
        {"percent": int, "message": str, "result": dict | None}
    执行失败请直接 raise，由 gRPC 层统一转成 FAILED 事件。
    """
    raise NotImplementedError
