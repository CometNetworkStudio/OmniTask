"""词达人 runner：题库 → LLM → 跳过，逐题推进并回写题库。"""

from __future__ import annotations

from core.bank import LocalBank
from core.config import Config
from core.events import event
from core.llm import LLMClient
from core.ratelimit import RateLimiter
from . import answer as answer_mod
from .client import CidarenClient, CidarenError, SecurityVerifyError

_MAX_QUESTIONS = 500
_CONFIG = Config.from_env()
_LIMITER = RateLimiter(
    concurrency=_CONFIG.platform_concurrency,
    min_interval=_CONFIG.request_min_interval,
    max_interval=_CONFIG.request_max_interval,
)


def _bank() -> LocalBank:
    return LocalBank(_CONFIG.bank_local_path)


def _progress(exam: dict, percent_hint: float) -> dict:
    if isinstance(exam, dict):
        done = exam.get("topic_done_num")
        total = exam.get("topic_total")
        if done and total:
            try:
                percent_hint = max(percent_hint, float(done) / float(total) * 100)
            except (TypeError, ZeroDivisionError):
                pass
    return event(int(percent_hint), f"进度 {int(percent_hint)}%")


def _run_task(client: CidarenClient, task: dict, llm, bank, params: dict):
    task_id = task.get("task_id") or task.get("id")
    task_type_int = int(task.get("task_type", params.get("task_type", 1)) or 1)
    task_kind = "ClassTask" if task_type_int == 2 else "StudyTask"
    course_id = task.get("course_id") or params.get("course_id")
    release_id = task.get("release_id") or params.get("release_id")
    list_id = task.get("list_id") or params.get("list_id")
    min_time = int(params.get("min_time", 5))
    max_time = int(params.get("max_time", 10))

    started = client.start_answer(task_id, task_kind, task_type_int, course_id=course_id, release_id=release_id)
    if started.get("done"):
        yield event(100, f"任务已完成({started.get('msg')})")
        return

    exam = started["exam"]
    total = exam.get("topic_total") or _MAX_QUESTIONS
    for index in range(_MAX_QUESTIONS):
        mode = exam.get("topic_mode")
        if mode in answer_mod.UNSUPPORTED_MODES:
            submission = None
        else:
            record = bank.lookup("cidaren", answer_mod.question_key(exam))
            submission = None
            if record:
                submission = answer_mod.resolve_from_record(exam, record["answer"])
            if submission is None:
                try:
                    submission = answer_mod.llm_answer(exam, llm) if llm.config.enabled else None
                except Exception:
                    submission = None

        if submission is None:
            step = client.skip_answer(task_kind, exam["topic_code"])
        else:
            result = client.verify_answer(task_kind, exam["topic_code"], submission)
            bank.save(answer_mod.derive_record(exam, result), overwrite=True)
            step = client.next_question(task_kind, result["topic_code"], min_time, max_time)

        yield _progress(exam, (index + 1) / total * 100)
        if step.get("done"):
            yield event(100, f"任务完成({step.get('msg')})")
            return
        exam = step["exam"]
    yield event(100, "达到本题上限，提前结束")


def run(action: str, params: dict, credentials: dict):
    token = credentials.get("token") or params.get("token")
    if not token:
        raise CidarenError("词达人需要 token 凭据")
    client = CidarenClient(token, _CONFIG, _LIMITER)
    llm = LLMClient(_CONFIG.llm)
    bank = _bank()
    try:
        user = client.token_check()
        yield event(1, f"token 校验通过 user={user.get('user_info', {}).get('nick_name', '')}".strip())
        if action in ("class", "class_task"):
            page = client.list_class_tasks(1)
            tasks = page.get("list") or page.get("data") or []
        else:
            tasks = client.list_study_tasks(params["course_id"]) if params.get("course_id") else []
        if not tasks:
            yield event(100, "未找到可执行任务")
            return
        for task in tasks:
            task.setdefault("course_id", params.get("course_id"))
            for evt in _run_task(client, task, llm, bank, params):
                yield evt
        yield event(100, "全部任务结束")
    except SecurityVerifyError as exc:
        raise CidarenError(str(exc)) from exc
    finally:
        bank.close()
