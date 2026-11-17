"""Export CI startup failure categories without raw test packets or bodies."""

import json
import os
from pathlib import Path
import re
import sys

if os.environ.get("GITHUB_ACTIONS") != "true":
    raise SystemExit("GitHub-only startup diagnosis")

source = Path(sys.argv[1])
destination = Path(sys.argv[2])
text = source.read_text(errors="replace") if source.exists() else ""
status = []
output = []
for line in text.splitlines():
    try:
        item = json.loads(line)
    except ValueError:
        output.append(line)
        continue
    if not isinstance(item, dict):
        continue
    if item.get("Action") in {"start", "run", "pass", "fail", "skip"}:
        status.append({key: item[key] for key in ["Action", "Package", "Test", "Elapsed"] if key in item})
    if isinstance(item.get("Output"), str):
        output.append(item["Output"])
combined = "\n".join(output)
lower = combined.lower()
markers = ["failed to init timezone", "must be set together", "invalid sub2api_test_redis_url",
           "docker is not available", "failed to start postgres container",
           "failed to start redis container", "failed to get postgres dsn",
           "failed to get redis host", "failed to get redis port", "failed to open sql db",
           "failed to apply db migrations", "failed to ping redis", "integration postgresql migrations complete",
           "segmentation violation", "sigsegv", "panic:", "connection refused",
           "context deadline exceeded", "permission denied", "too many open files",
           "no space left on device", "pull access denied", "rate limit", "unauthorized",
           "internal_error", "unexpected eof", "no such image", "not found",
           "unhealthy", "could not connect", "failed to connect", "container exited",
           "cannot connect to the docker daemon", "docker api", "client version", "minimum supported api",
           "timeout", "timed out", "undefined:", "build failed"]
matched = [marker for marker in markers if marker in lower]
safe_errors = []
for line in output:
    if not any(marker in line.lower() for marker in markers):
        continue
    # Keep only bootstrap diagnostics. Protocol payload/test assertion output
    # stays in the runner's temporary file and is never exported.
    if not any(marker in line.lower() for marker in markers[:12]):
        continue
    line = re.sub(r"(?:postgres(?:ql)?|redis|https?)://\S+", "[address]", line)
    line = re.sub(r"(?i)(password|token|secret|key)\s*[=:]\s*[^\s,;]+", r"\1=[redacted]", line)
    line = re.sub(r"[0-9a-f]{32,}", "[identifier]", line)
    safe_errors.append(line.strip()[:1500])
result = {"application_sha": os.environ.get("APP_SHA"), "source_exists": source.exists(),
          "source_bytes": len(text.encode()), "matched_error_categories": matched,
          "bootstrap_errors": safe_errors[:8], "test_status": status,
          "raw_packets_or_bodies_exported": False}
destination.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
print(json.dumps(result, ensure_ascii=False))
