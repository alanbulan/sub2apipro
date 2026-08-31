# Daily Upstream Sync

The checker fetches the upstream remote, writes `.codex-upstream-sync/report.md`, and returns the pending commit count.

```bash
git remote add upstream https://github.com/Wei-Shaw/sub2api.git
./scripts/check-upstream.sh
```

Run a manual Codex analysis:

```bash
codex exec "Use the upstream-sync skill. Read .codex-upstream-sync/report.md and analyze/merge the pending upstream updates without modifying protected paths."
```

Install a daily 04:00 China Standard Time (`Asia/Shanghai`) job:

```cron
0 15,16 * * * [ "$(TZ=Asia/Shanghai date +\%H)" = 04 ] && /sub2api/deploy/cron/upstream-sync
```

The production server uses the `America/New_York` timezone and its Debian cron
does not support `CRON_TZ`. The guard runs the wrapper only when the current
hour in `Asia/Shanghai` is `04`, covering both daylight-saving and standard
time without a manual schedule change.

Required configuration:

- `upstream` remote must point to the source repository;
- `custom/protected-paths.txt` lists files and directories that must not be overwritten;
- `codex` must be available on `PATH` for scheduled execution;
- logs are written under `.codex-upstream-sync/logs/`.

The scheduled wrapper is designed for the 4GB production server. When upstream commits are pending, it runs Codex in a systemd transient service with bounded memory, swap, process count, and runtime. It permits only low-cost static checks; Docker builds, frontend production builds, backend tests, Go tests, TypeScript compilation, package-manager scripts, and development servers are left to GitHub Actions. The wrapper pushes approved changes only to `origin/main`, never to `upstream`.

An interrupted, failed, or blocked review does not advance `.codex-upstream-sync/last-seen-head`. The next scheduled run will review the same upstream commits. A successful `noop` review advances the state only after the complete review proves that no code change is required.
