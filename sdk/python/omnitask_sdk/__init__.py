"""OmniTask Script SDK (Python)。

提供 stdio JSON 运行时、动作注册、网络/限流/题库等能力，供平台脚本接入。

两种接入方式：
    # A. 生成器式
    from omnitask_sdk import serve
    def run(action, params, credentials):
        yield {"percent": 10, "message": "..."}
    serve(run)

    # B. 装饰器式
    from omnitask_sdk import action, serve
    @action("study")
    def study(ctx):
        ctx.progress(10, "...")
    serve()
"""

from .context import AdapterContext
from .decorator import action, registered_actions
from .listing import listing, registered_listings
from .runtime import serve

__all__ = [
    "AdapterContext",
    "action",
    "registered_actions",
    "listing",
    "registered_listings",
    "serve",
]
