"""AdapterContext：传给脚本动作的上下文（参数、凭据、进度上报）。"""

from __future__ import annotations


class AdapterContext:
    """动作函数接收的上下文对象。

    - ``ctx.params`` / ``ctx.credentials``：来自任务的参数与凭据
    - ``ctx.progress(percent, message)``：上报进度
    - ``ctx.log(message)``：上报一条日志（沿用当前进度）
    - ``ctx.result``：最终结果（脚本可写入，随 succeeded 事件返回）
    - ``ctx.canceled``：进程被终止时为 True（若脚本处理信号）
    """

    def __init__(self, params: dict, credentials: dict, emit, action: str = "") -> None:
        self.params = dict(params or {})
        self.credentials = dict(credentials or {})
        self.action = action
        self.result: dict = {}
        self.canceled = False
        self._emit = emit
        self._last_percent = 0

    def progress(self, percent: int, message: str = "", result: dict | None = None) -> None:
        self._last_percent = int(percent)
        self._emit(
            {"percent": self._last_percent, "message": str(message), "result": result or {}}
        )

    def log(self, message: str) -> None:
        self._emit({"percent": self._last_percent, "message": str(message), "result": {}})
