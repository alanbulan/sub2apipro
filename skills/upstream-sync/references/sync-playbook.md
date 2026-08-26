# Conflict Playbook

## Protected Themes

`frontend/src/styles/themes/*.css` are independent skins. If upstream edits Tailwind palette values, do not restore hardcoded teal colors. Keep Tailwind mapped to CSS variables and add or adjust variables in each affected theme file. Preserve the five IDs: `minimalism`, `neoBrutalism`, `apple`, `notion`, and `wabiSabi`.

If upstream introduces a new global color role:

1. add the semantic variable to all five theme files;
2. map it in `frontend/tailwind.config.js`;
3. replace only the specific upstream UI usage with the semantic class or variable;
4. run `pnpm build` and visually verify light and dark mode.

## Branding

The fork intentionally removes visible `Sub2API` and official GitHub links. OAuth provider labels and protocol identifiers such as `github_oauth_*` must remain because they are functional integrations. Site name comes from settings and may be empty before initialization; do not reintroduce product-name fallbacks.

If upstream changes a user-visible string containing the upstream name, translate it to neutral wording such as "this platform", "platform", or "API gateway". Internal keys, storage keys, cache prefixes, and wire fields should remain compatible unless the user approves a migration.

## Update Sources

Upstream release/update logic points at the original project. Preserve the mechanism but treat repository/image URLs as deployment configuration if upstream adds configuration support. Do not hardcode the old official links back into UI copy.

## Automation Files

`skills/upstream-sync/**`, `scripts/check-upstream.sh`, and `custom/protected-paths.txt` define this fork's workflow. Prefer rebasing local improvements over accepting an upstream version of these files.
