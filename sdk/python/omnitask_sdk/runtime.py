"""stdio NDJSON 运行时（Host ↔ 脚本）。

两种接入方式：

1) 旧式（生成器）：

    def run(action, params, credentials):
        yield {"percent": 10, "message": "..."}
    serve(run)

2) 装饰器式（推荐）：

    @action("study")
    def study(ctx):
        ctx.step("login", "登录", "running")
        ctx.progress(10, "...")
    serve()

事件 v2 在旧字段之外增加 step / log / metric 等组件，旧脚本无需改动。
"""

from __future__ import annotations

import json
import sys
import threading

from .context import AdapterContext
from .decorator import registered_actions
from .listing import registered_listings

# 允许透传到 Host 的可选字段。
_EXTRA_KEYS = ("current", "total", "unit", "stage", "step", "label", "status", "level", "key", "value")


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
        cancel_event = threading.Event()
        _start_cancel_watcher(cancel_event)
        _emit({"type": "event", "task_id": task_id, "event": "started", "message": "开始执行"})
        try:
            if callable(target):
                result = _run_legacy(target, action_id, params, credentials, task_id)
            else:
                result = _run_action(handlers, action_id, params, credentials, task_id, cancel_event)
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


def _run_action(handlers, action_id, params, credentials, task_id, cancel_event=None) -> dict:
    if handlers is None or action_id not in handlers:
        raise KeyError(f"未注册的 action: {action_id}")
    ctx = AdapterContext(
        params,
        credentials,
        lambda event: _emit_progress(task_id, event),
        action=action_id,
        cancel_event=cancel_event,
    )
    outcome = handlers[action_id](ctx)
    if hasattr(outcome, "__iter__") and not isinstance(outcome, (dict, str, bytes)):
        for event in outcome:
            _emit_progress(task_id, event if isinstance(event, dict) else {"message": str(event)})
    elif isinstance(outcome, dict) and not ctx.result:
        ctx.result.update(outcome)
    return ctx.result


def _start_cancel_watcher(cancel_event: threading.Event) -> None:
    def watch() -> None:
        for line in sys.stdin:
            line = line.strip()
            if not line:
                continue
            try:
                message = json.loads(line)
            except Exception:
                continue
            if message.get("type") == "cancel":
                cancel_event.set()
                return

    threading.Thread(target=watch, daemon=True).start()


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
    payload = {
        "type": "event",
        "task_id": task_id,
        "event": event.get("event") or "progress",
        "percent": int(percent) if percent is not None else 0,
        "message": str(event.get("message", "")),
        "result": event.get("result") or {},
    }
    for key in _EXTRA_KEYS:
        if key in event and event[key] not in (None, ""):
            payload[key] = event[key]
    _emit(payload)


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
