"""CLI 桥：把已有命令行项目「零代码」接入 OmniTask。

在脚本仓库只写 ``omnitask.json``（无需自己的 Python 代码）：

    {
      "id": "some-cli", "name": "某 CLI 项目", "runtime": "python3",
      "module": "omnitask_sdk.bridge",
      "actions": [
        {"id": "run", "name": "运行", "command": ["mycli", "run", "--course", "{cid}"]}
      ],
      "credentials": [{"key": "token", "label": "Token", "type": "password"}],
      "params": [{"key": "cid", "label": "课程 id", "type": "text", "default": ""}]
    }

Host 会以 cwd=脚本目录、``python -m omnitask_sdk.bridge`` 启动本桥；
本桥读取 ``./omnitask.json`` 找到对应 ``command`` 执行，并把子进程输出转为事件。
命令行参数支持 ``{key}`` 占位符（取自 params / credentials / action / task_id）。

进度：默认识别形如 ``[progress] 42 正在处理`` 的输出行；可用 manifest 的
``progress_regex`` 自定义（需含捕获组 1=百分比、2=消息）。

组件标记（可选，零代码也能用）：

    [progress] 42 正在处理
    [step] login running 登录
    [metric] correct 正确率 96 %
    [log] warn 遇到限流，稍后重试

其余输出按 info 日志展示。
"""

from __future__ import annotations

import json
import os
import re
import signal
import subprocess
import sys

MANIFEST = "omnitask.json"

DEFAULT_PROGRESS_RE = re.compile(r"\[progress\]\s*(\d+)\s*(.*)")
STEP_RE = re.compile(r"\[step\]\s*(\S+)\s+(running|succeeded|failed|skipped)\s*(.*)")
METRIC_RE = re.compile(r"\[metric\]\s*(\S+)\s*(\S*)\s*(\S+)\s*(\S*)")
LOG_RE = re.compile(r"\[log\]\s*(debug|info|warn|error)\s+(.*)")


def serve() -> None:
    request = _read_request()
    if request is None:
        return
    task_id = request.get("task_id", "")
    action_id = request.get("action", "")
    params = request.get("params") or {}
    credentials = request.get("credentials") or {}

    manifest = _load_manifest()
    action = _find_action(manifest, action_id)
    if action is None:
        _emit_failed(task_id, f"manifest 中未找到 action: {action_id}")
        return
    command = action.get("command")
    if not command:
        _emit_failed(task_id, f"action {action_id} 未配置 command")
        return

    argv = [_expand(part, params, credentials, task_id, action_id) for part in command]
    regex = _compile(manifest.get("progress_regex"))

    try:
        proc = subprocess.Popen(argv, stdout=subprocess.PIPE, text=True, bufsize=1)
    except FileNotFoundError as exc:
        _emit_failed(task_id, f"命令不存在: {exc}")
        return

    def _terminate(signum, frame):  # noqa: ANN001
        proc.terminate()

    signal.signal(signal.SIGTERM, _terminate)

    _emit({"type": "event", "task_id": task_id, "event": "started", "message": "开始执行"})

    percent = 0
    last_line = ""
    assert proc.stdout is not None
    for raw in proc.stdout:
        line = raw.rstrip("\n")
        if not line.strip():
            continue
        last_line = line
        step = STEP_RE.search(line)
        if step:
            _emit(
                {
                    "type": "event",
                    "task_id": task_id,
                    "event": "step",
                    "step": step.group(1),
                    "status": step.group(2),
                    "label": (step.group(3) or step.group(1)).strip(),
                }
            )
            continue
        metric = METRIC_RE.search(line)
        if metric:
            _emit(
                {
                    "type": "event",
                    "task_id": task_id,
                    "event": "metric",
                    "key": metric.group(1),
                    "label": metric.group(2) or metric.group(1),
                    "value": metric.group(3),
                    "unit": metric.group(4),
                }
            )
            continue
        log_line = LOG_RE.search(line)
        if log_line:
            _emit(
                {
                    "type": "event",
                    "task_id": task_id,
                    "event": "log",
                    "level": log_line.group(1),
                    "message": log_line.group(2),
                }
            )
            continue
        match = regex.search(line) if regex else None
        if match:
            percent = int(match.group(1))
            message = (match.group(2) or "").strip() or line
            _emit_progress(task_id, percent, message)
        else:
            _emit(
                {
                    "type": "event",
                    "task_id": task_id,
                    "event": "log",
                    "level": "info",
                    "message": line,
                }
            )

    code = proc.wait()
    if code == 0:
        _emit({"type": "event", "task_id": task_id, "event": "succeeded", "percent": 100, "message": "done", "result": {}})
    else:
        _emit_failed(task_id, f"命令退出码 {code}" + (f"：{last_line}" if last_line else ""))


def _read_request() -> dict | None:
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            return json.loads(line)
        except Exception:
            _emit_failed("", "invalid request json")
            return None
    return None


def _load_manifest() -> dict:
    roots = []
    if os.environ.get("OMNITASK_SCRIPT_DIR"):
        roots.append(os.environ["OMNITASK_SCRIPT_DIR"])
    roots.append(os.getcwd())
    for root in roots:
        path = os.path.join(root, MANIFEST)
        try:
            with open(path, encoding="utf-8-sig") as handle:
                return json.load(handle)
        except Exception:
            continue
    return {}


def _find_action(manifest: dict, action_id: str) -> dict | None:
    for action in manifest.get("actions") or []:
        if action.get("id") == action_id:
            return action
    return None


def _expand(part: str, params: dict, credentials: dict, task_id: str, action_id: str) -> str:
    context = {"action": action_id, "task_id": task_id, **params, **credentials}

    def repl(match: re.Match) -> str:
        return str(context.get(match.group(1), match.group(0)))

    return re.sub(r"\{(\w+)\}", repl, part)


def _compile(pattern: str | None):
    if not pattern:
        return DEFAULT_PROGRESS_RE
    try:
        return re.compile(pattern)
    except re.error:
        return DEFAULT_PROGRESS_RE


def _emit_progress(task_id: str, percent: int, message: str) -> None:
    _emit(
        {
            "type": "event",
            "task_id": task_id,
            "event": "progress",
            "percent": int(percent),
            "message": str(message),
            "result": {},
        }
    )


def _emit_failed(task_id: str, message: str) -> None:
    _emit({"type": "event", "task_id": task_id, "event": "failed", "percent": 0, "message": message, "result": {}})


def _emit(obj: dict) -> None:
    sys.stdout.write(json.dumps(obj, ensure_ascii=False) + "\n")
    sys.stdout.flush()


if __name__ == "__main__":
    serve()
