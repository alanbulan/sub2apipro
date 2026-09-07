# Conflict Playbook

## Protected Themes

`frontend/src/styles/themes/*.css` contains 24 independent skins. Use the full
`appThemes` registry in `frontend/src/composables/useAppTheme.ts` as the source
of theme IDs and stylesheet names. Preserve all 24, their imports in
`frontend/src/style.css`, switcher/preview controls, locale labels, and saved
preferences. The old five-theme list is obsolete. Keep Tailwind mapped to CSS
variables rather than restoring upstream hardcoded palette values.

If upstream introduces a new global color role:

1. account for the semantic variable across all 24 theme files;
2. map it in `frontend/tailwind.config.js`;
3. replace only the specific upstream UI usage with the semantic class or variable;
4. in a development worktree, run `pnpm build` and visually verify light and dark mode.

On the production sync server, including any worktree on that machine, use
static text checks only. GitHub Actions validates the theme registry, each CSS
file's existence and selector, and the global imports via the critical frontend
test suite. Production build/test commands never belong in a sync-host step.

## Branding

The fork intentionally removes visible `Sub2API` and official GitHub links. OAuth provider labels and protocol identifiers such as `github_oauth_*` must remain because they are functional integrations. Site name comes from settings and may be empty before initialization; do not reintroduce product-name fallbacks.

If upstream changes a user-visible string containing the upstream name, translate it to neutral wording such as "this platform", "platform", or "API gateway". Internal keys, storage keys, cache prefixes, and wire fields should remain compatible unless the user approves a migration.

## Update Sources

Upstream release/update logic points at the original project. Preserve the mechanism but treat repository/image URLs as deployment configuration if upstream adds configuration support. Do not hardcode the old official links back into UI copy.

## Automation Files

`skills/upstream-sync/**`, `scripts/check-upstream.sh`, and `custom/protected-paths.txt` define this fork's workflow. Prefer rebasing local improvements over accepting an upstream version of these files. Deployment manifests and helpers are upstream-owned unless they are explicitly listed in `custom/protected-paths.txt`.

The wrapper saves the candidate/base/upstream SHAs in `candidate.json` before
push, and the exact CI run URL/state in `ci-status.json`. A GitHub API timeout,
network error, or rate limit means the CI result is unknown. Honor `Retry-After`
and `X-RateLimit-Reset`, then retry within the deadline. On expiry retain the
checkpoint and resume it on the next invocation. Never advance the upstream
baseline without a confirmed CI success and verified promotion. If main or the
candidate changed, prepare a fresh review instead of promoting the stale result.

Protected files are source-control requirements, not filesystem `chattr +i`
flags. Deployment pulls the image built by Actions; compiled/minified assets
need not have the same bytes as their source. Preserve source customizations
and runtime behavior; do not compare generated bundles with source-file hashes.

## Ent Schema and Custom Backend Modules

An Ent runtime panic such as `interface {} is int, not string` can result from
generated descriptor indexes lagging behind schema changes. Review
`backend/ent/schema/`, generated entities, `ent/runtime/runtime.go`, and
`ent/migrate/schema.go` together. Preserve fork schema additions and check each
affected runtime index against the field name, type, default, and validator in
the merged schema. Do not resolve a generated-file conflict by blindly taking
all upstream output when the fork has extra fields.

Run code generation only on a GitHub runner when needed, using the repository's
`go generate ./ent` and `go generate ./cmd/server` commands. Review the generated
diff before committing it; never paper over the panic with a type assertion
change. CI first exercises gateway initialization and capture regression, then
runs the full unit/integration suites. Adding a custom module also requires
retaining its DI wiring, routes, schema/migration, and handler tests as upstream
interfaces evolve. The presence of a migration is not itself a blocker.

## Renamed Locale Keys

Follow upstream locale key moves at every call site while preserving our visible
wording. For example, `admin.settings.siteNamePlaceholder` moved to
`admin.settings.site.siteNamePlaceholder`; stale references fail the locale-key
CI test even if the translation text still exists. Keep both language trees
consistent. Do not restore upstream branding to satisfy the test or remove the
locale completeness check. Read the frontend log artifact for the exact key.

## Gateway Model-Allowlist Conflict

When upstream introduces `groupModelAllowlist` on gateway routes, preserve the
single-handler `rootRoute` helper. The upstream route-coverage test requires
this exact middleware order and a `handler gin.HandlerFunc` parameter:

```go
rootRoute := func(method, path string, limit gin.HandlerFunc, handler gin.HandlerFunc) {
	r.Handle(method, path, limit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), groupModelAllowlist, compositeTarget, requireGroupAnthropic, handler)
}
```

Do not change `rootRoute` to accept a variadic handler chain. To retain fork
conversation capture on root Responses and Chat Completions routes, pass a
single handler created with `h.Gateway.ConversationCaptureWithHandler(protocol,
handler)`. This keeps capture around the endpoint execution while retaining
the allowlist between authentication and composite routing. Keep the same
capture middleware on Codex Responses and Gemini generation routes.
