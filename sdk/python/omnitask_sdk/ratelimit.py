"""平台级并发与请求间隔限流。

- 平台并发默认 2（全可配），同一账号串行。
- Agent 以线程池运行，故用 threading。
"""

from __future__ import annotations

import contextlib
import os
import random
import sqlite3
import threading
import time
from pathlib import Path


class SharedPacer:
    """跨进程共享的请求间隔，使用 Host 提供的 SQLite 文件做轻量协调。"""

    def __init__(self, path: str) -> None:
        Path(path).parent.mkdir(parents=True, exist_ok=True)
        self._lock = threading.Lock()
        self._conn = sqlite3.connect(path, timeout=5, check_same_thread=False)
        self._conn.execute("PRAGMA journal_mode=WAL")
        self._conn.execute("PRAGMA busy_timeout=5000")
        self._conn.execute(
            "CREATE TABLE IF NOT EXISTS rate_pace (platform TEXT PRIMARY KEY, next_at REAL NOT NULL)"
        )
        self._conn.commit()

    def close(self) -> None:
        with self._lock:
            self._conn.close()

    def wait(self, platform: str, min_interval: float, max_interval: float) -> None:
        interval = random.uniform(min_interval, max_interval)
        with self._lock:
            try:
                self._conn.execute("BEGIN IMMEDIATE")
                row = self._conn.execute(
                    "SELECT next_at FROM rate_pace WHERE platform = ?", (platform,)
                ).fetchone()
                now = time.time()
                next_at = max(now, float(row[0]) if row else now)
                self._conn.execute(
                    "INSERT INTO rate_pace(platform, next_at) VALUES(?, ?) "
                    "ON CONFLICT(platform) DO UPDATE SET next_at = excluded.next_at",
                    (platform, next_at + interval),
                )
                self._conn.commit()
            except Exception:
                self._conn.rollback()
                raise
        wait = next_at - now
        if wait > 0:
            time.sleep(wait)


class RateLimiter:
    def __init__(
        self,
        concurrency: int = 2,
        min_interval: float = 0.8,
        max_interval: float = 2.5,
        shared_path: str | None = None,
    ) -> None:
        self._concurrency = max(1, concurrency)
        self._min_interval = max(0.0, min_interval)
        self._max_interval = max(self._min_interval, max_interval)
        self._guard = threading.Lock()
        self._semaphores: dict[str, threading.Semaphore] = {}
        self._account_locks: dict[tuple[str, str], threading.Lock] = {}
        self._last_request: dict[str, float] = {}
        self._shared = None
        path = shared_path or os.environ.get("OMNITASK_RATE_DB", "")
        if path:
            try:
                self._shared = SharedPacer(path)
            except Exception:
                self._shared = None

    def _semaphore(self, platform: str) -> threading.Semaphore:
        with self._guard:
            if platform not in self._semaphores:
                self._semaphores[platform] = threading.Semaphore(self._concurrency)
            return self._semaphores[platform]

    def account_lock(self, platform: str, account: str) -> threading.Lock:
        with self._guard:
            key = (platform, account)
            if key not in self._account_locks:
                self._account_locks[key] = threading.Lock()
            return self._account_locks[key]

    def _pace(self, platform: str) -> None:
        if self._shared is not None:
            try:
                self._shared.wait(platform, self._min_interval, self._max_interval)
                return
            except Exception:
                # 共享库异常时退回进程内节流，避免影响任务执行。
                pass
        with self._guard:
            last = self._last_request.get(platform)
            now = time.monotonic()
            wait = 0.0
            if last is not None:
                wait = random.uniform(self._min_interval, self._max_interval) - (now - last)
            self._last_request[platform] = now + max(0.0, wait)
        if wait > 0:
            time.sleep(wait)

    @contextlib.contextmanager
    def platform_slot(self, platform: str):
        """占用平台并发额度，并按随机间隔节流。"""
        sem = self._semaphore(platform)
        sem.acquire()
        try:
            self._pace(platform)
            yield
        finally:
            sem.release()

    @contextlib.contextmanager
    def account_slot(self, platform: str, account: str):
        """同一账号串行 + 平台并发 + 节流。"""
        with self.account_lock(platform, account):
            with self.platform_slot(platform):
                yield
