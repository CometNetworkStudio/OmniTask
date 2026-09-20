import importlib
import os
import sys

_HERE = os.path.dirname(os.path.abspath(__file__))
_SDK = os.environ.get("OMNITASK_SDK") or os.path.dirname(os.path.dirname(_HERE))
if _SDK and _SDK not in sys.path:
    sys.path.insert(0, _SDK)
_SCRIPTS = os.path.dirname(_HERE)
if _SCRIPTS not in sys.path:
    sys.path.insert(0, _SCRIPTS)

from runtime import serve  # noqa: E402

run = importlib.import_module(f"{os.path.basename(_HERE)}.runner").run  # noqa: E402

if __name__ == "__main__":
    serve(run)
