"""本地命令行验证入口（不经 Go 后端）。

示例：
    agent/.venv/bin/python run_cli.py --platform cidaren --action study \
        --param course_id=JJ_3 --credential token=xxxx
    agent/.venv/bin/python run_cli.py --platform welearn --action time \
        --credential username=u --credential password=p --param duration=60

凭据也可来自环境变量：OMNITASK_CIDAREN_TOKEN、OMNITASK_WELEARN_USERNAME/PASSWORD。
"""

from __future__ import annotations

import argparse
import importlib
import json
import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
sys.path.insert(0, str(ROOT / "gen"))
sys.path.insert(0, str(ROOT))

from core.config import load_dotenv  # noqa: E402

_ENV_CREDENTIALS = {
    "cidaren": {"token": "OMNITASK_CIDAREN_TOKEN"},
    "welearn": {"username": "OMNITASK_WELEARN_USERNAME", "password": "OMNITASK_WELEARN_PASSWORD"},
}


def _kv(items):
    result = {}
    for item in items or []:
        if "=" not in item:
            raise SystemExit(f"参数格式应为 k=v: {item}")
        key, value = item.split("=", 1)
        result[key] = value
    return result


def main() -> int:
    load_dotenv()
    parser = argparse.ArgumentParser(description="OmniTask agent CLI")
    parser.add_argument("--platform", required=True, choices=sorted(_ENV_CREDENTIALS))
    parser.add_argument("--action", required=True)
    parser.add_argument("--param", action="append", default=[], help="任务参数 k=v，可重复")
    parser.add_argument("--credential", action="append", default=[], help="凭据 k=v，可重复")
    args = parser.parse_args()

    credentials = _kv(args.credential)
    for key, env_name in _ENV_CREDENTIALS[args.platform].items():
        if key not in credentials:
            value = os.environ.get(env_name)
            if value:
                credentials[key] = value

    runner = importlib.import_module(f"scripts.{args.platform}.runner")
    params = _kv(args.param)
    try:
        for event in runner.run(args.action, params, credentials):
            print(json.dumps(event, ensure_ascii=False))
    except Exception as exc:  # noqa: BLE001
        print(f"ERROR {type(exc).__name__}: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
