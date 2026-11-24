"""OmniTask 脚本开发命令行。

用法（在脚本目录执行）：

    python -m omnitask_sdk.cli check
    python -m omnitask_sdk.cli run --action study --param cid=2416
    python -m omnitask_sdk.cli run --action study --request request.json
    python -m omnitask_sdk.cli schema

安装 SDK 后也可直接使用 ``omnitask-script``。
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
from pathlib import Path

from . import __version__

MANIFEST = "script.json"
ALLOWED_STATUS = {"running", "succeeded", "failed", "skipped"}


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="omnitask-script", description="OmniTask 脚本开发工具")
    sub = parser.add_subparsers(dest="command", required=True)

    check = sub.add_parser("check", help="校验 script.json 与入口文件")
    check.add_argument("dir", nargs="?", default=".", help="脚本目录，默认当前目录")

    run = sub.add_parser("run", help="本地模拟 Host 执行一次脚本")
    run.add_argument("dir", nargs="?", default=".", help="脚本目录，默认当前目录")
    run.add_argument("--action", default="", help="动作 id")
    run.add_argument("--param", action="append", default=[], help="参数 k=v，可重复")
    run.add_argument("--credential", action="append", default=[], help="凭据 k=v，可重复")
    run.add_argument("--request", help="直接使用请求 JSON 文件")
    run.add_argument("--task-id", default="local-dev", help="模拟任务 id")
    run.add_argument("--timeout", type=float, default=0, help="超时秒数，0 表示不限")

    sub.add_parser("schema", help="输出 script.json 的 JSON Schema")

    args = parser.parse_args(argv)
    if args.command == "check":
        return _check(Path(args.dir))
    if args.command == "run":
        return _run(args)
    if args.command == "schema":
        return _print_schema()
    return 1


def _load_manifest(root: Path) -> dict:
    path = root / MANIFEST
    if not path.exists():
        raise SystemExit(f"未找到 {path}")
    try:
        return json.loads(path.read_text(encoding="utf-8-sig"))
    except Exception as exc:  # noqa: BLE001
        raise SystemExit(f"解析 {path} 失败: {exc}") from exc


def _check(root: Path) -> int:
    problems: list[str] = []
    try:
        manifest = _load_manifest(root)
    except SystemExit as exc:
        print(f"[FAIL] {exc}")
        return 1

    protocol = int(manifest.get("protocol") or 1)
    if protocol > 2:
        problems.append(f"protocol={protocol} 高于当前支持版本 2")
    if protocol < 2:
        print("[HINT] 建议设置 \"protocol\": 2，以启用动作级参数与组件事件")
    for key in ("id", "runtime"):
        if not manifest.get(key):
            problems.append(f"缺少 {key}")
    if not manifest.get("entry") and not manifest.get("module"):
        problems.append("需要 entry 或 module")
    if manifest.get("entry") and not (root / manifest["entry"]).exists():
        problems.append(f"entry 文件不存在: {manifest['entry']}")

    actions = manifest.get("actions") or []
    if protocol >= 2 and not actions:
        problems.append("协议 v2 至少需要一个 action")
    seen = set()
    for action in actions:
        aid = action.get("id")
        if not aid:
            problems.append("存在缺少 id 的 action")
            continue
        if aid in seen:
            problems.append(f"action id 重复: {aid}")
        seen.add(aid)
        _check_fields(action.get("params") or [], f"action {aid} 的 params", problems)
        result = action.get("result")
        if result:
            if result.get("type") not in (None, "table", "links", "keyvalue"):
                problems.append(f"action {aid} 的 result.type 不支持: {result.get('type')}")
            _check_fields(result.get("columns") or [], f"action {aid} 的 result.columns", problems)

    _check_fields(manifest.get("params") or [], "params", problems)
    _check_fields(manifest.get("credentials") or [], "credentials", problems)
    discovery = manifest.get("discovery")
    if discovery:
        _check_fields(discovery.get("columns") or [], "discovery.columns", problems)
        for aid in discovery.get("actions") or []:
            if aid not in seen:
                problems.append(f"discovery.actions 引用了不存在的 action: {aid}")

    if problems:
        for item in problems:
            print(f"[FAIL] {item}")
        return 1
    print(f"[OK] {manifest.get('name') or manifest.get('id')} 校验通过（protocol={protocol}）")
    return 0


def _check_fields(fields: list[dict], scope: str, problems: list[str]) -> None:
    seen = set()
    for field in fields:
        key = field.get("key")
        if not key:
            problems.append(f"{scope} 存在空 key")
            continue
        if key in seen:
            problems.append(f"{scope} key 重复: {key}")
        seen.add(key)
        if field.get("type") == "select" and not field.get("options"):
            problems.append(f"{scope}.{key} 是 select，但缺少 options")


def _run(args) -> int:
    root = Path(args.dir).resolve()
    try:
        manifest = _load_manifest(root)
    except SystemExit as exc:
        print(f"[FAIL] {exc}", file=sys.stderr)
        return 1

    if args.request:
        request = json.loads(Path(args.request).read_text(encoding="utf-8-sig"))
        request.setdefault("type", "execute")
        request.setdefault("task_id", args.task_id)
    else:
        params = _pairs(args.param)
        credentials = _pairs(args.credential)
        request = {
            "type": "execute",
            "task_id": args.task_id,
            "platform": manifest.get("id", ""),
            "action": args.action or (manifest.get("actions") or [{}])[0].get("id", ""),
            "params": params,
            "credentials": credentials,
            "bank_addr": os.environ.get("OMNITASK_BANK_ADDR", ""),
        }

    runtime = manifest.get("runtime") or "python3"
    if str(runtime).startswith("python"):
        runtime = sys.executable
    command = [runtime]
    if manifest.get("module"):
        command += ["-m", manifest["module"]]
    else:
        command.append(manifest.get("entry"))
    command += manifest.get("args") or []

    print(f"[RUN] {' '.join(command)}", file=sys.stderr)
    print(f"[REQ] {json.dumps(request, ensure_ascii=False)}", file=sys.stderr)
    env = dict(os.environ)
    sdk_root = str(Path(__file__).resolve().parent.parent)
    env.setdefault("PYTHONPATH", sdk_root)
    env.setdefault("OMNITASK_SDK", sdk_root)
    try:
        proc = subprocess.run(
            command,
            cwd=root,
            input=json.dumps(request, ensure_ascii=False) + "\n",
            text=True,
            capture_output=True,
            env=env,
            timeout=args.timeout or None,
        )
    except subprocess.TimeoutExpired:
        print("[FAIL] 脚本执行超时", file=sys.stderr)
        return 1

    code = 0
    for line in proc.stdout.splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            print(f"[BAD] 非 JSON 输出: {line}", file=sys.stderr)
            code = 1
            continue
        print(_format_event(event))
        if event.get("event") == "failed":
            code = 1
    if proc.stderr.strip():
        print("[stderr]", file=sys.stderr)
        print(proc.stderr.rstrip(), file=sys.stderr)
    if proc.returncode != 0:
        code = proc.returncode
    return code


def _format_event(event: dict) -> str:
    kind = event.get("event") or event.get("type") or "?"
    if kind == "progress":
        progress = f"{event.get('percent', 0)}%"
        if event.get("total"):
            progress += f" {event.get('current', 0)}/{event.get('total')}{event.get('unit', '')}"
        return f"[progress] {progress} {event.get('message', '')}".rstrip()
    if kind == "log":
        return f"[{event.get('level', 'info')}] {event.get('message', '')}"
    if kind == "step":
        return f"[step:{event.get('status', '')}] {event.get('label') or event.get('step', '')}"
    if kind == "metric":
        return f"[metric] {event.get('label') or event.get('key', '')}={event.get('value', '')}{event.get('unit', '')}"
    return f"[{kind}] {event.get('message', '')}".rstrip()


def _pairs(items: list[str]) -> dict[str, str]:
    out = {}
    for item in items:
        if "=" not in item:
            raise SystemExit(f"参数格式应为 k=v: {item}")
        key, value = item.split("=", 1)
        out[key] = value
    return out


def _print_schema() -> int:
    path = Path(__file__).with_name("script.schema.json")
    print(path.read_text(encoding="utf-8"))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
