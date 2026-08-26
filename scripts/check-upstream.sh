#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
UPSTREAM="${UPSTREAM_REMOTE:-upstream}"
BRANCH="${UPSTREAM_BRANCH:-main}"
REPORT_DIR="$ROOT/.codex-upstream-sync"
STATE_FILE="$REPORT_DIR/last-seen-head"
LEGACY_STATE_FILE="$REPORT_DIR/last-seen"
REPORT="$REPORT_DIR/report.md"

mkdir -p "$REPORT_DIR"
cd "$ROOT"

git fetch --quiet "$UPSTREAM" "$BRANCH"
REMOTE_HEAD="$(git rev-parse --verify "$UPSTREAM/$BRANCH^{commit}")"

is_valid_baseline() {
  local candidate="$1"

  [[ "$candidate" =~ ^[0-9a-fA-F]{40}$ ]] || return 1
  git cat-file -e "$candidate^{commit}" 2>/dev/null || return 1
  git merge-base --is-ancestor "$candidate" "$REMOTE_HEAD"
}

LAST_HEAD=""
if [[ -s "$STATE_FILE" ]]; then
  CANDIDATE="$(tr -d '[:space:]' < "$STATE_FILE")"
  if is_valid_baseline "$CANDIDATE"; then
    LAST_HEAD="$CANDIDATE"
  fi
fi

# The local branch is the safest recovery baseline when the state file is
# missing or was written by an older runner after an incomplete sync.
if [[ -z "$LAST_HEAD" ]]; then
  LAST_HEAD="$(git merge-base HEAD "$REMOTE_HEAD" 2>/dev/null || true)"
  if ! is_valid_baseline "$LAST_HEAD"; then
    LAST_HEAD=''
  fi
fi

if [[ -z "$LAST_HEAD" && -s "$REPORT" ]]; then
  CANDIDATE="$(sed -n 's/^Previous head: `\([^`]*\)`.*/\1/p' "$REPORT" | head -n 1)"
  if is_valid_baseline "$CANDIDATE"; then
    LAST_HEAD="$CANDIDATE"
  fi
fi

if [[ -z "$LAST_HEAD" ]]; then
  echo 'Unable to determine a safe upstream baseline.' >&2
  echo 'Set .codex-upstream-sync/last-seen-head to a verified upstream commit and retry.' >&2
  exit 1
fi

COUNT="$(git rev-list --count "$LAST_HEAD..$REMOTE_HEAD")"

cat > "$REPORT" <<EOF
# Upstream sync report

Generated at: $(date -Is)
Remote: $UPSTREAM/$BRANCH
Remote head: \`$REMOTE_HEAD\`
Previous head: \`${LAST_HEAD:-none}\`
Pending commits: $COUNT

Legacy state file: \`$LEGACY_STATE_FILE\` is ignored; only \`last-seen-head\` is authoritative.

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
