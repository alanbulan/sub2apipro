#!/bin/bash
# Runner-only diagnostics; keep full logs in artifacts and a readable excerpt.
set -eu
for ci_log in "$@"; do
  [ -s "$ci_log" ] || continue
  excerpt="$(grep -E -C 4 '(^--- FAIL:|^FAIL[[:space:]]|panic:|error TS|Error:|AssertionError|Error Trace:)' "$ci_log" | tail -n 100 || true)"
  if [ -z "$excerpt" ]; then excerpt="$(tail -n 60 "$ci_log")"; fi
  excerpt="$(printf '%s' "$excerpt" | head -c 10000)"
  {
    printf '\n### %s\n\n```text\n' "${ci_log##*/}"
    printf '%s\n' "$excerpt"
    printf '```\n\nFull output is available in the run artifacts.\n'
  } >> "$GITHUB_STEP_SUMMARY"
  excerpt="${excerpt//%/%25}"
  excerpt="${excerpt//$'\r'/%0D}"
  excerpt="${excerpt//$'\n'/%0A}"
  printf '::error title=CI failure details::%s\n' "$excerpt"
done
