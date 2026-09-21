"""stdio NDJSON 运行时（Host ↔ 脚本）。

两种接入方式：

1) 旧式（生成器）：

    def run(action, params, credentials):
        yield {"percent": 10, "message": "..."}
    serve(run)

2) 装饰器式：

    @action("study")
    def study(ctx):
        ctx.progress(10, "...")
    serve()
"""

from __future__ import annotations

import json
import sys

from .context import AdapterContext
from .decorator import registered_actions
from .listing import registered_listings


def serve(target=None) -> None:
    """从 stdin 读一条请求（execute 或 discover）并处理，结果写 stdout。

    target：可选。传入 ``run(action, params, credentials)`` 生成器即旧式；
    不传则使用 ``@action`` 注册的动作。discover 使用 ``@listing`` 注册的函数。
    """
    handlers = None if callable(target) else registered_actions()
    listers = registered_listings()
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            request = json.loads(line)
        except Exception:
            _raw_error("", "invalid request json")
            return
        request_type = request.get("type")
        if request_type == "discover":
            try:
                _run_discover(listers, request)
            except Exception as exc:  # noqa: BLE001
                _raw_error("", f"{type(exc).__name__}: {exc}")
            return
        if request_type != "execute":
            continue
        task_id = request.get("task_id", "")
        action_id = request.get("action", "")
        params = request.get("params") or {}
        credentials = request.get("credentials") or {}
        try:
            if callable(target):
                result = _run_legacy(target, action_id, params, credentials, task_id)
            else:
                result = _run_action(handlers, action_id, params, credentials, task_id)
            _emit(
                {
                    "type": "event",
                    "task_id": task_id,
                    "event": "succeeded",
                    "percent": 100,
                    "message": "done",
                    "result": result or {},
                }
            )
        except Exception as exc:  # noqa: BLE001
            _raw_error(task_id, f"{type(exc).__name__}: {exc}")
        return


def _run_legacy(run_func, action_id, params, credentials, task_id) -> dict:
    for event in run_func(action_id, params, credentials):
        _emit_progress(task_id, event)
    return {}


def _run_action(handlers, action_id, params, credentials, task_id) -> dict:
    if handlers is None or action_id not in handlers:
        raise KeyError(f"未注册的 action: {action_id}")
    ctx = AdapterContext(
        params, credentials, lambda event: _emit_progress(task_id, event), action=action_id
    )
    outcome = handlers[action_id](ctx)
    if hasattr(outcome, "__iter__") and not isinstance(outcome, (dict, str, bytes)):
        for event in outcome:
            _emit_progress(task_id, event if isinstance(event, dict) else {"message": str(event)})
    return ctx.result


def _run_discover(listers, request) -> None:
    platform = request.get("platform", "")
    params = request.get("params") or {}
    credentials = request.get("credentials") or {}
    func = listers.get("default") or next(iter(listers.values()), None)
    if func is None:
        raise RuntimeError("未注册发现函数（请用 @listing 注册）")
    ctx = AdapterContext(params, credentials, lambda _ev: None, action="discover")
    outcome = func(ctx)
    if isinstance(outcome, dict) and "items" in outcome:
        raw_items = outcome["items"]
    elif hasattr(outcome, "__iter__") and not isinstance(outcome, (dict, str, bytes)):
        raw_items = list(outcome)
    else:
        raw_items = outcome or []
    items = [_normalize_item(item) for item in raw_items]
    _emit({"type": "items", "platform": platform, "items": items})


def _normalize_item(item) -> dict:
    if not isinstance(item, dict):
        return {"key": str(item), "fields": {"value": str(item)}}
    if isinstance(item.get("fields"), dict):
        key = str(item.get("key") or item.get("id") or "")
        fields = {str(k): _str(v) for k, v in item["fields"].items()}
        return {"key": key, "fields": fields}
    key = str(item.get("key") or item.get("id") or item.get("course_id") or "")
    fields = {str(k): _str(v) for k, v in item.items() if k != "key"}
    return {"key": key, "fields": fields}


def _str(value) -> str:
    return "" if value is None else str(value)


def _emit_progress(task_id: str, event: dict) -> None:
    percent = event.get("percent")
    _emit(
        {
            "type": "event",
            "task_id": task_id,
            "event": "progress",
            "percent": int(percent) if percent is not None else 0,
            "message": str(event.get("message", "")),
            "result": event.get("result") or {},
        }
    )


def _raw_error(task_id: str, message: str) -> None:
    _emit(
        {
            "type": "event",
            "task_id": task_id,
            "event": "failed",
            "percent": 0,
            "message": message,
            "result": {},
        }
    )


def _emit(obj: dict) -> None:
    sys.stdout.write(json.dumps(obj, ensure_ascii=False) + "\n")
    sys.stdout.flush()
