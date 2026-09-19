"""WeLearn runner：刷时长 / 提交进度。"""

from __future__ import annotations

import time

from core.config import Config
from core.events import event
from core.ratelimit import RateLimiter
from .client import WeLearnClient, WeLearnError

_CONFIG = Config.from_env()
_LIMITER = RateLimiter(
    concurrency=_CONFIG.platform_concurrency,
    min_interval=_CONFIG.request_min_interval,
    max_interval=_CONFIG.request_max_interval,
)


def _is_actionable(item, include_completed: bool) -> bool:
    if not isinstance(item, dict):
        return True
    if str(item.get("isvisible")) == "false":
        return False
    if include_completed:
        return True
    return "未" in str(item.get("iscomplete", ""))


def _sco_id(item) -> str | None:
    if isinstance(item, dict):
        for key in ("scoid", "sco", "sco_id", "id"):
            if item.get(key) not in (None, ""):
                return str(item[key])
        return None
    return str(item)


def _pick_course(courses: list, want) -> dict:
    if not courses:
        raise WeLearnError("账号下没有课程")
    if want in (None, ""):
        return courses[0]
    for course in courses:
        if str(course.get("cid")) == str(want) or str(course.get("id")) == str(want):
            return course
    raise WeLearnError(f"未找到课程 cid={want}")


def run(action: str, params: dict, credentials: dict):
    username = credentials.get("username") or params.get("username")
    password = credentials.get("password") or params.get("password")
    if not username or not password:
        raise WeLearnError("WeLearn 需要账号密码凭据")

    client = WeLearnClient(username, password, _CONFIG, _LIMITER)
    message = client.login()
    yield event(2, message)

    courses = client.get_courses()
    yield event(5, f"课程数 {len(courses)}")
    course = _pick_course(courses, params.get("cid"))
    cid = course.get("cid") or course.get("id")
    info = client.get_course_info(cid)
    uid, classid = info["uid"], info["classid"]

    include_completed = str(params.get("include_completed", "")).lower() in ("1", "true", "yes")
    if params.get("unit_idx") not in (None, ""):
        sco_items = client.get_sco_leaves(cid, uid, classid, params["unit_idx"])
    else:
        sco_items = []
        for unit in info["units"]:
            unit_idx = unit.get("unitidx") or unit.get("idx") or unit.get("id")
            if unit_idx is None:
                continue
            sco_items.extend(client.get_sco_leaves(cid, uid, classid, unit_idx))

    actionable = [item for item in sco_items if _is_actionable(item, include_completed)]
    scoids = [sid for sid in (_sco_id(item) for item in actionable) if sid]
    limit = params.get("limit")
    if limit:
        scoids = scoids[: int(limit)]
    if not scoids:
        yield event(100, "没有未完成的 SCO")
        return
    yield event(8, f"待处理 SCO 数 {len(scoids)}")

    duration = int(params.get("duration", 60))
    accuracy = params.get("accuracy", "100")
    for index, scoid in enumerate(scoids, start=1):
        base = (index - 1) / len(scoids) * 92 + 8
        if action == "progress":
            client.submit_course_progress(cid, uid, classid, scoid, accuracy)
            yield event(int(base + 92 / len(scoids)), f"SCO {scoid} 进度已提交")
            continue
        client.start_sco(cid, uid, scoid)
        for elapsed in range(1, duration + 1):
            time.sleep(1)
            if elapsed % 60 == 0:
                client.keep_sco(cid, uid, scoid)
            if elapsed % 10 == 0:
                within = elapsed / duration * (92 / len(scoids))
                yield event(int(base + within), f"SCO {scoid} 已挂 {elapsed}s/{duration}s")
        client.save_sco(cid, uid, scoid)
        yield event(int(base + 92 / len(scoids)), f"SCO {scoid} 完成")
    yield event(100, "全部 SCO 处理完成")
