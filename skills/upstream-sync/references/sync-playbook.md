# Conflict Playbook

## Protected Themes

`frontend/src/styles/themes/*.css` are independent skins. If upstream edits Tailwind palette values, do not restore hardcoded teal colors. Keep Tailwind mapped to CSS variables and add or adjust variables in each affected theme file. Preserve the five IDs: `minimalism`, `neoBrutalism`, `apple`, `notion`, and `wabiSabi`.

If upstream introduces a new global color role:

1. add the semantic variable to all five theme files;
2. map it in `frontend/tailwind.config.js`;
3. replace only the specific upstream UI usage with the semantic class or variable;
4. in a development worktree, run `pnpm build` and visually verify light and dark mode.

On the production sync server, do not run that build or any other local test.
GitHub Actions is the validation authority there.

## Branding

The fork intentionally removes visible `Sub2API` and official GitHub links. OAuth provider labels and protocol identifiers such as `github_oauth_*` must remain because they are functional integrations. Site name comes from settings and may be empty before initialization; do not reintroduce product-name fallbacks.

If upstream changes a user-visible string containing the upstream name, translate it to neutral wording such as "this platform", "platform", or "API gateway". Internal keys, storage keys, cache prefixes, and wire fields should remain compatible unless the user approves a migration.

## Update Sources

Upstream release/update logic points at the original project. Preserve the mechanism but treat repository/image URLs as deployment configuration if upstream adds configuration support. Do not hardcode the old official links back into UI copy.

## Automation Files

`skills/upstream-sync/**`, `scripts/check-upstream.sh`, and `custom/protected-paths.txt` define this fork's workflow. Prefer rebasing local improvements over accepting an upstream version of these files. Deployment manifests and helpers are upstream-owned unless they are explicitly listed in `custom/protected-paths.txt`.

The candidate branch is a durable CI checkpoint. A GitHub API timeout, network
error, or rate limit means the CI result is unknown, not that it failed. The
wrapper must retain the candidate and retry status queries through its deadline;
only an explicit non-success CI conclusion or the configured deadline may stop
promotion.

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
