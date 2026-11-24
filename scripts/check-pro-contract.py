#!/usr/bin/env python3
"""Static source/CI guard. No dependency install, test or build is performed."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

args_parser = argparse.ArgumentParser()
args_parser.add_argument('--ref')
args_parser.add_argument('--deployment', action='store_true')
args = args_parser.parse_args()
root = Path(__file__).resolve().parents[1]
def read(path):
    if args.ref:
        return subprocess.check_output(['git', 'show', f'{args.ref}:{path}'], cwd=root).decode()
    return (root / path).read_text()

def require(condition, message):
    if not condition:
        raise SystemExit(message)

manifest = json.loads(read('custom/canonical-upstream.json'))
require(manifest['upstream']['repository'] == 'Wei-Shaw/sub2api', 'Canonical upstream changed')
require(manifest['upstream']['branch'] == 'main', 'Canonical upstream branch changed')
ci = read('.github/workflows/backend-ci.yml')
for job in ['shell', 'test', 'frontend', 'golangci-lint', 'release-helpers', 'pro', 'security']:
    require(f'  {job}:\n' in ci, f'Missing current CI job: {job}')
for command in ['make test-unit', 'make test-integration', 'make test-frontend']:
    require(command in ci, f'Missing canonical check: {command}')
pro = read('.github/workflows/pro-ci.yml')
for command in ['TestCanonicalMigration', 'rememberMePro.spec.ts', 'pnpm build', 'upstream-sync-ci.test.cjs']:
    require(command in pro, f'Missing Pro validation: {command}')
archive = json.loads(read('custom/legacy-migrations/manifest.json'))
for migration in archive['migrations']:
    if 'archive' in migration:
        require(hashlib.sha256(read(migration['archive']).encode()).hexdigest() == migration['sha256'],
                f"Modified historical migration: {migration['path']}")
# Dedicated sources must survive even when upstream renames shared entry points.
for path in ['backend/internal/service/auth_feishu_auto.go', 'backend/internal/service/feishu_key_api.go',
             'backend/internal/service/intelligence_schedule.go', 'backend/internal/service/conversation_log.go',
             'frontend/src/views/user/PoolDashboardView.vue', 'frontend/src/utils/authStorage.ts']:
    require(bool(read(path)), f'Missing Pro feature: {path}')
# Only the current reviewed merge may clear this gate, immediately before an
# owner-authorized production promotion. Branch checks do not need this flag.
if args.deployment:
    require(manifest['deployment']['status'] == 'approved', 'Migration deployment awaits owner review')
print('Canonical source, immutable SQL and current Pro CI contracts are intact.')
