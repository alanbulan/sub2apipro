#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
UPSTREAM="${UPSTREAM_REMOTE:-upstream}"
BRANCH="${UPSTREAM_BRANCH:-main}"
REPORT_DIR="$ROOT/.codex-upstream-sync"
STATE_FILE="$REPORT_DIR/last-seen-head"
REPORT="$REPORT_DIR/report.md"

mkdir -p "$REPORT_DIR"
cd "$ROOT"

git fetch "$UPSTREAM" "$BRANCH" >/dev/null 2>&1
REMOTE_HEAD="$(git rev-parse "$UPSTREAM/$BRANCH")"

LAST_HEAD=""
if [[ -s "$STATE_FILE" ]]; then
  LAST_HEAD="$(tr -d '[:space:]' < "$STATE_FILE")"
  git cat-file -e "$LAST_HEAD^{commit}" 2>/dev/null || LAST_HEAD=''
fi

if [[ -z "$LAST_HEAD" ]]; then
  LAST_HEAD="$(git rev-parse "$REMOTE_HEAD~10" 2>/dev/null || true)"
fi

if [[ -n "$LAST_HEAD" ]]; then
  COMMITS=$(git rev-list "$LAST_HEAD..$REMOTE_HEAD")
else
  COMMITS=$(git rev-list -10 "$REMOTE_HEAD")
fi

if [[ -n "$COMMITS" ]]; then
  COUNT=$(printf '%s\n' "$COMMITS" | wc -l | tr -d ' ')
else
  COUNT=0
fi

cat > "$REPORT" <<EOF
# Upstream sync report

Generated at: $(date -Is)
Remote: $UPSTREAM/$BRANCH
Remote head: \`$REMOTE_HEAD\`
Previous head: \`${LAST_HEAD:-none}\`
Pending commits: $COUNT

## Changed files

EOF

if [[ "$COUNT" -eq 0 ]]; then
  echo 'No new upstream commits.' >> "$REPORT"
else
  git diff --name-status "${LAST_HEAD:-$(git rev-parse "$REMOTE_HEAD"^$COUNT)}" "$REMOTE_HEAD" >> "$REPORT"
  cat >> "$REPORT" <<'EOF'

## Codex instruction

Invoke the `upstream-sync` skill with this report. Analyze every pending commit before changing files.
EOF
fi

printf '%s\n' "$COUNT"
