"""AdapterContext：传给脚本动作的上下文（参数、凭据、进度、日志、指标、结果）。

面向脚本开发者的“组件”都挂在这里：

- ``ctx.progress``    进度条（兼容旧写法）
- ``ctx.progress_of`` 按 current/total 上报进度
- ``ctx.step``        步骤清单
- ``ctx.log``         日志面板
- ``ctx.metric``      指标卡片（如正确率、剩余数量）
- ``ctx.result``      最终结果键值
"""

from __future__ import annotations

import threading


class CancelledError(RuntimeError):
    """平台请求取消当前任务。"""


class AdapterContext:
    """动作函数接收的上下文对象。

    - ``ctx.params`` / ``ctx.credentials``：来自任务的参数与凭据
    - ``ctx.progress(percent, message)``：上报百分比进度
    - ``ctx.progress_of(current, total, message, unit)``：上报可计算进度的数量
    - ``ctx.step(key, label, status)``：上报一个步骤的状态
    - ``ctx.log(message, level)``：上报一条日志
    - ``ctx.metric(key, label, value, unit)``：上报一个指标
    - ``ctx.result``：最终结果（脚本可写入，随 succeeded 事件返回）
    """

    def __init__(
        self,
        params: dict,
        credentials: dict,
        emit,
        action: str = "",
        cancel_event: threading.Event | None = None,
    ) -> None:
        self.params = dict(params or {})
        self.credentials = dict(credentials or {})
        self.action = action
        self.result: dict = {}
        self._cancel_event = cancel_event or threading.Event()
        self._emit = emit
        self._last_percent = 0

    @property
    def canceled(self) -> bool:
        return self._cancel_event.is_set()

    def check_canceled(self) -> None:
        if self.canceled:
            raise CancelledError("任务已取消")

    def progress(
        self,
        percent: int,
        message: str = "",
        result: dict | None = None,
        *,
        current: int | None = None,
        total: int | None = None,
        unit: str = "",
        stage: str = "",
    ) -> None:
        """上报百分比进度；percent 会被限制在 0-100。"""
        value = max(0, min(100, int(percent)))
        self._last_percent = value
        payload = {
            "percent": value,
            "current": _int_or_zero(current),
            "total": _int_or_zero(total),
            "unit": str(unit or ""),
            "stage": str(stage or ""),
            "message": str(message),
            "result": result or {},
        }
        self._emit(payload)

    def progress_of(
        self,
        current: int,
        total: int,
        message: str = "",
        *,
        unit: str = "",
        stage: str = "",
    ) -> None:
        """按 current/total 上报进度，百分比由平台计算。"""
        current = max(0, int(current))
        total = max(0, int(total))
        if total <= 0:
            self.log("progress_of 需要 total > 0，已忽略本次进度", "warn")
            return
        percent = int(current * 100 / total)
        self.progress(
            percent,
            message,
            current=current,
            total=total,
            unit=unit,
            stage=stage,
        )

    def step(self, key: str, label: str = "", status: str = "running") -> None:
        """上报步骤状态：running / succeeded / failed / skipped。"""
        status = str(status or "running")
        if status not in ("running", "succeeded", "failed", "skipped"):
            status = "running"
        self._emit(
            {
                "event": "step",
                "step": str(key),
                "label": str(label or key),
                "status": status,
            }
        )

    def log(self, message: str, level: str = "info") -> None:
        """上报一条日志：debug / info / warn / error。"""
        level = str(level or "info").lower()
        if level not in ("debug", "info", "warn", "error"):
            level = "info"
        self._emit({"event": "log", "level": level, "message": str(message)})

    def debug(self, message: str) -> None:
        self.log(message, "debug")

    def warn(self, message: str) -> None:
        self.log(message, "warn")

    def error(self, message: str) -> None:
        self.log(message, "error")

    def metric(self, key: str, label: str, value, unit: str = "") -> None:
        """上报一个指标，前端按 key 保留最新值。"""
        self._emit(
            {
                "event": "metric",
                "key": str(key),
                "label": str(label or key),
                "value": str(value),
                "unit": str(unit or ""),
            }
        )


def _int_or_zero(value) -> int:
    try:
        return max(0, int(value))
    except (TypeError, ValueError):
        return 0
