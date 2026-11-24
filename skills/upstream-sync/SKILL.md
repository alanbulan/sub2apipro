---
name: upstream-sync
description: Follow Wei-Shaw/sub2api main while preserving Pro branding, themes, authentication, Feishu, employee key delivery, conversation logs, account pool and IQ tests.
---

# Canonical upstream synchronization

The only update source is `https://github.com/Wei-Shaw/sub2api.git`, branch `main`.
The writable distribution repository is `alanbulan/sub2apipro`. The upstream push
URL must remain `DISABLED`. Never merge `ranxi2001/production` again.

Read `custom/canonical-upstream.json`, `custom/protected-paths.txt`,
`docs/canonical-upstream-migration.md` and `.codex-upstream-sync/report.md`.
Review every pending canonical commit, create a local backup ref, and merge the
exact reported SHA. Preserve these Pro behaviors while accepting canonical fixes:

- All 24 themes, saved preferences, appearance controls, white-label UI and layout.
- Remember-me storage: unchecked sessions stay in sessionStorage; checked sessions
  persist. Plaintext passwords belong only to the browser password manager.
- Independent conversation records, bounded HTTP/SSE/WS capture and existing data.
- Tenant-restricted Feishu login, QR setup/callback, ordinary-user provisioning,
  names/avatars, directory sync and real enterprise-mailbox backfill. Imported
  profile mail is not proof of ownership for password/email authentication.
- Bulk recharge, one reusable employee Key, one initial gift, previewed mail jobs,
  and admin-JWT single/batch issuance API that does not send mail.
- User-scoped account pool, account names/quota and shared diagnostic modals.
  Account credentials, emails, proxies and upstream error bodies remain private.
- Manual and scheduled IQ questions/probes, bounded execution and history. Do not
  bring back old smart-operations, automatic penalties, ticket harvesting or bots.

Protected dedicated files stay byte-identical in unattended updates. Shared
wiring, schemas, routes, translations and auth storage callers must merge both
canonical changes and Pro behavior. CI definitions may change existing lines to
follow current upstream, provided `scripts/check-pro-contract.py` still accepts
required canonical and Pro checks. There is no append-only historical CI rule.

Never run tests, builds, dependency installs, code generation, typecheck or dev
servers on this host, including worktrees. Use source inspection, formatting,
`git diff --check`, `bash -n` and `sh -n` only. All executable verification belongs
to GitHub Actions. Never mutate production data to rehearse an upgrade.

Preserve applied SQL content/checksums and existing ledger entries. Retired fork
SQL lives in `custom/legacy-migrations/` outside the embedded migration list.
New migrations are additive and must pass both fresh-install and pinned legacy
upgrade rehearsals. Do not remove data tables or bypass checksum checks.

For unattended maintenance, the wrapper owns candidate pushes and promotion.
Write `analysis.md` and `result.json` (`applied`, `noop` or `blocked`, exact
`remote_head`). Never push main yourself. Candidate CI must pass for its exact
SHA before the wrapper promotes it. Unknown, canceled, failed or timed-out CI is
never success; preserve the checkpoint for retry or repair.

The initial canonical migration separately requires the owner's deployment
review. Keep its deployment manifest unapproved until that explicit approval.
Follow session authorization for later ordinary updates; do not introduce another
approval gate merely because upstream changes authentication or database code.

See [the playbook](references/sync-playbook.md) for shared-code merge boundaries.
