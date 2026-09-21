"""动作注册（装饰器风格）。

用法：
    from omnitask_sdk import action, serve

    @action("study")
    def study(ctx):
        ctx.progress(10, "开始")
        ...
        ctx.result["score"] = 100

    if __name__ == "__main__":
        serve()          # 从 stdio 接收 execute 并按 action 分发
"""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

_ACTIONS: dict[str, Callable[[Any], Any]] = {}


def action(name: str):
    def deco(fn: Callable[[Any], Any]) -> Callable[[Any], Any]:
        _ACTIONS[name] = fn
        setattr(fn, "__omnitask_action__", name)
        return fn

    return deco


def registered_actions() -> dict[str, Callable[[Any], Any]]:
    return dict(_ACTIONS)
