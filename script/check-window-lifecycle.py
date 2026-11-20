#!/usr/bin/env python3
"""Exercise native window close, Dock reopen and quit in an isolated app."""
from pathlib import Path
import shutil
import subprocess
import tempfile

root = Path(__file__).resolve().parent.parent
with tempfile.TemporaryDirectory(prefix="gitviz-window-check-") as directory:
    target = Path(directory)
    for source in root.glob("*.go"):
        if source.name != "main.go" and not source.name.endswith("_test.go"):
            shutil.copy2(source, target / source.name)
    for name in ("go.mod", "go.sum"):
        shutil.copy2(root / name, target / name)
    shutil.copy2(root / "script/window-lifecycle-smoke.go.txt", target / "main.go")
    binary = target / "gitviz-lifecycle-check"
    subprocess.run(["go", "build", "-o", str(binary), "."], cwd=target, check=True)
    subprocess.run([str(binary)], cwd=target, check=True, timeout=25)
