"""发现（discover）注册：列出账号在平台上的条目，供 OmniTask 前端选择。

用法：
    from omnitask_sdk import listing, serve

    @listing()
    def courses(ctx):
        # 返回条目列表；每项可为：
        #   {"id": "...", "title": "...", ...}           （扁平，runtime 自动包装）
        #   {"key": "...", "fields": {"...": "..."}}     （显式 key+fields）
        return my_client.list_courses()

    if __name__ == "__main__":
        serve()
"""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

_LISTERS: dict[str, Callable[[Any], Any]] = {}


def listing(name: str = "default"):
    def deco(fn: Callable[[Any], Any]) -> Callable[[Any], Any]:
        _LISTERS[name] = fn
        return fn

    return deco


def registered_listings() -> dict[str, Callable[[Any], Any]]:
    return dict(_LISTERS)
