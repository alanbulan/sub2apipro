"""Adapt independent interfaces in disposable historical CI checkouts.

This is runner-only source preparation. It never changes regression assertions
or reports a test result. Each adaptation prints its final source hash; existing
RED validators still reject compilation errors and require named failures.
"""
from pathlib import Path
import os
import re
import subprocess
import sys

source_root = Path(os.environ.get('SYNC_CI_SOURCE_ROOT', Path(__file__).resolve().parent.parent)).resolve()
root = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
# RED fixtures may be git worktrees or plain `git archive` directories.
if (root / 'go.mod').is_file() and root.name == 'backend':
    root = root.parent
if not (root / 'backend/go.mod').is_file():
    raise RuntimeError('historical fixture must contain backend/go.mod: ' + str(root))
source_ref = os.environ.get('SYNC_CI_SOURCE_REF') or os.environ.get('EXPECTED_HEAD_SHA') or 'HEAD'
source_ref = subprocess.check_output(
    ['git', '-C', str(source_root), 'rev-parse', '--verify', source_ref + '^{commit}'], text=True
).strip()


def current(path):
    # Some RED jobs replace files in the same checkout that owns this helper.
    # Read the committed candidate, not those restored historical files.
    return subprocess.check_output(
        ['git', '-C', str(source_root), 'show', source_ref + ':' + path], text=True
    )


def read(path):
    p = root / path
    return p.read_text() if p.is_file() else ''


def write(path, source):
    p = root / path
    if p.is_file() and p.read_text() == source:
        return
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(source)
    sha = subprocess.check_output(['git', '-C', str(root), 'hash-object', path], text=True).strip()
    print(f'historical independent interface: {path} final_blob={sha}', flush=True)


def field(path, struct, name, kind):
    source = read(path)
    match = re.search(r'type ' + struct + r' struct \{\n(.*?)\n\}', source, re.S)
    if match and not re.search(r'^\s*' + name + r'\s', match[1], re.M):
        write(path, source[:match.start(1)] + '\t' + name + ' ' + kind + '\n' + source[match.start(1):])


def declaration(source, name):
    # All requested declarations are top-level, with braces at column zero.
    match = re.search(r'^(?:func(?:\s+\([^\n]+?\))?|type|const|var)\s+' + re.escape(name) + r'\b[^\n]*', source, re.M)
    if not match:
        raise RuntimeError('missing current declaration: ' + name)
    end = match.end()
    line = match.group()
    if (line.startswith('func ') or '{' in line) and not line.rstrip().endswith('}'):
        closing = re.search(r'^\}', source[end:], re.M)
        if not closing:
            raise RuntimeError('unterminated current declaration: ' + name)
        end += closing.end()
    return source[match.start():end].rstrip() + '\n'


def append(path, code, imports):
    source = read(path)
    source += '\n' + code
    for package in imports:
        if '"' + package + '"' not in source:
            source = source.replace('import (\n', 'import (\n\t"' + package + '"\n', 1)
    write(path, source)


# Full upstream RED overlays remove fork additions while retaining the fork's
# tests. Supply only the independent Feishu/capture leaves and their declarations;
# keep every restored native recovery implementation and regression unchanged.
feishu_tests = root / 'backend/internal/pkg/feishu'
if any(feishu_tests.glob('*_test.go')) and not (feishu_tests / 'client.go').is_file():
    for path in (
        'backend/internal/pkg/feishu/client.go',
        'backend/internal/pkg/feishu/directory.go',
        'backend/internal/service/setting_feishu.go',
        'backend/internal/service/auth_feishu_auto.go',
        'backend/internal/service/feishu_setup.go',
        'backend/internal/service/feishu_directory.go',
        'backend/internal/service/conversation_log.go',
        'backend/internal/repository/feishu_setup_store.go',
        'backend/internal/repository/feishu_user.go',
        'backend/internal/repository/conversation_log_repo.go',
        'backend/internal/handler/auth_feishu_oauth.go',
        'backend/internal/handler/admin/setting_feishu.go',
        'backend/internal/handler/admin/conversation_log_handler.go',
        'backend/internal/handler/conversation_capture.go',
        'backend/internal/handler/conversation_capture_ws.go',
    ):
        if not read(path):
            write(path, current(path))
    path = 'backend/internal/pkg/authguard/context.go'
    if 'func WithoutRegistrationIP(' not in read(path):
        append(path, declaration(current(path), 'WithoutRegistrationIP'), [])
    path = 'backend/internal/service/domain_constants.go'
    if 'const FeishuConnectSyntheticEmailDomain ' not in read(path):
        append(path, declaration(current(path), 'FeishuConnectSyntheticEmailDomain'), [])
    field('backend/internal/service/user_service.go', 'UserIdentitySummarySet', 'Feishu', 'UserIdentitySummary `json:"feishu"`')
    field('backend/internal/handler/admin/setting_handler.go', 'SettingHandler', 'feishuSetup', '*service.FeishuSetupService')
    field('backend/internal/handler/admin/setting_handler.go', 'SettingHandler', 'feishuDirectory', '*service.FeishuDirectoryService')
    path = 'backend/internal/handler/auth_handler.go'
    field(path, 'AuthHandler', 'feishuClient', '*feishu.Client')
    append(path, '', ['github.com/Wei-Shaw/sub2api/internal/pkg/feishu'])
    field('backend/internal/handler/gateway_handler.go', 'GatewayHandler', 'conversationLogService', '*service.ConversationLogService')
    path = 'backend/internal/handler/auth_oauth_pending_flow.go'
    if 'func setOAuthPendingBrowserCookieWithMaxAge(' not in read(path):
        append(path, declaration(current(path), 'setOAuthPendingBrowserCookieWithMaxAge'), [])


# Later Wire injection references a fork-only field absent in old handler blobs.
# Wire is not part of these endpoint RED oracles; keep the tested handler exact.
handler = read('backend/internal/handler/openai_gateway_handler.go')
wire_path = 'backend/internal/handler/wire.go'
if handler and not re.search(r'^\s*conversationLogService\s', handler, re.M):
    wire = read(wire_path)
    start = wire.find('func ProvideOpenAIGatewayHandler(')
    end = wire.find('\nfunc ', start + 1)
    if start >= 0:
        before, block, after = wire[:start], wire[start:end if end >= 0 else len(wire)], wire[end:] if end >= 0 else ''
        write(wire_path, before + block.replace('\th.conversationLogService = conversationLogService\n', '') + after)

# Snapshot/TTL tests added after Redis 6.0's original compatibility oracle need
# an independent constructor and sentinel. The restored Redis commands remain.
path = 'backend/internal/repository/concurrency_cache.go'
source = read(path)
if source and 'func NewConcurrencyCacheWithAPIKeyTTL(' not in source:
    field(path, 'concurrencyCache', 'apiKeySlotTTLSeconds', 'int')
    append(path, declaration(current(path), 'NewConcurrencyCacheWithAPIKeyTTL'), [])
path = 'backend/internal/repository/scheduler_cache.go'
if read(path) and 'schedulerEmptySnapshotMember =' not in read(path):
    append(path, 'const schedulerEmptySnapshotMember = "__empty__"\n', [])

# Preserve old deletion semantics for later, unselected automatic callers.
path = 'backend/internal/repository/scheduled_test_repo.go'
source = read(path)
if source and 'func deleteScheduledTestPlanTx(' not in source:
    match = re.search(r'func \(r \*scheduledTestPlanRepository\) Delete\(.*?\n\}', source, re.S)
    if match and '\tstate, err := readQualityState' in match.group():
        body = match.group().split('\tstate, err := readQualityState', 1)[1]
        body = '\tstate, err := readQualityState' + body
        body = body.replace('return tx.Commit()', 'return nil')
        append(path, 'func deleteScheduledTestPlanTx(ctx context.Context, tx *sql.Tx, id int64) error {\n' + body, [])

# Shared batches gained a transaction-aware result writer. Reuse the old
# writer's body without changing its existing Create or cooldown behavior.
source = read(path)
if source and 'func insertScheduledTestResult(' not in source:
    match = re.search(r'func \(r \*scheduledTestResultRepository\) Create\([^\n]+\{\n(.*?)\n\}', source, re.S)
    if match:
        body = match[1].replace('r.db.QueryRowContext(', 'q.QueryRowContext(')
        append(path, 'func insertScheduledTestResult(ctx context.Context, q planRowQuerier, result *service.ScheduledTestResult) (*service.ScheduledTestResult, error) {\n' + body + '\n}\n', [])

# A later management helper must not repair the restored list/create oracle.
path = 'backend/internal/service/openai_auth_connections.go'
has_management_compat = any('func (s *OpenAIOAuthReauthService) managementConnectionAccount(' in p.read_text() for p in (root / 'backend/internal/service').glob('*.go'))
if read(path) and not has_management_compat:
    append(path, '''func (s *OpenAIOAuthReauthService) managementConnectionAccount(ctx context.Context, id int64) (*Account, error) {
    return s.connectionAccount(ctx, id)
}
''', [])

path = 'backend/internal/pkg/requesttiming/content.go'
field(path, 'ContentObservation', 'SSEEventFirstMS', '*float64 `json:"sse_event_first_ms"`')
field(path, 'ContentObservation', 'SemanticFirstMS', '*float64 `json:"semantic_first_ms"`')

# The later independent scheduler calls gained a legacy-account predicate.
# Drop it only when the restored scheduling implementation has no such input;
# retain the restored selection body and its scoped-feedback RED assertions.
path = 'backend/internal/service/openai_gateway_scheduling.go'
signature = re.search(r'^func \(s \*OpenAIGatewayService\) selectAccountWithLoadAwareness\([^\n]+', read(path), re.M)
if signature and 'stickyBindingAllowed' not in signature[0]:
    path = 'backend/internal/service/openai_account_scheduler.go'
    source = read(path)
    source = source.replace(', useUpstreamTokenCost, acceptLegacyAccount)', ', useUpstreamTokenCost)')
    if source:
        write(path, source)

# Restore only the independent memo context and scan-result shape around old
# compatibility pipeline logic; it still does not implement the missing fix.
path = 'backend/internal/service/openai_gateway_cc_pipeline.go'
source = read(path)
needs_compat_context = 'openAICompatTempUnschedContext(' in read('backend/internal/service/openai_gateway_upstream_errors.go')
if source and needs_compat_context and 'func openAICompatTempUnschedContext(' not in source:
    code = 'const openAICompatTempUnschedAttemptKey = "openai_compat_temp_unsched_attempt"\n'
    code += declaration(current(path), 'openAICompatTempUnschedAttempt')
    code += declaration(current(path), 'openAICompatTempUnschedContext')
    append(path, code, ['context'])
field(path, 'ccStreamScanState', 'SemanticFailure', '[]byte')

# Current independent stream/trigger callers coexist with the old negative-cache
# implementation. Keep every old method; only supply missing declarations.
path = 'backend/internal/service/ratelimit_service.go'
source = read(path)
needs_memo = 'bindTempUnschedAttempt(' in read('backend/internal/service/openai_account_runtime_block_fastpath.go') or 'tempUnschedAttemptMemo' in read('backend/internal/service/openai_gateway_cc_pipeline.go')
if source and needs_memo and 'func bindTempUnschedAttempt(' not in source:
    latest = current(path)
    old_struct = re.search(r'type RateLimitService struct \{\n(.*?)\n\}', source, re.S)
    new_struct = re.search(r'type RateLimitService struct \{\n(.*?)\n\}', latest, re.S)
    if old_struct and new_struct:
        for name, kind in re.findall(r'^\s*(\w+)\s+([^\n/]+?)\s*$', new_struct[1], re.M):
            field(path, 'RateLimitService', name, kind)
        source = read(path)
        if 'sync.' in source and '"sync"' not in source:
            write(path, source.replace('import (\n', 'import (\n\t"sync"\n', 1))
            source = read(path)
    starts = list(re.finditer(r'^(func|type|const|var)\b[^\n]*', latest, re.M))
    pieces = []
    for i, match in enumerate(starts):
        block = latest[match.start():starts[i + 1].start() if i + 1 < len(starts) else len(latest)]
        line = match.group()
        # Identify named declarations and grouped constants by all their names.
        if line.startswith('func '):
            name = re.search(r'^func (?:\([^\n]+?\) )?(\w+)', line).group(1)
            exists = re.search(r'^func (?:\([^\n]+?\) )?' + name + r'\b', source, re.M)
        elif re.match(r'^(?:const|var) \(', line):
            names = re.findall(r'^\s*(\w+)\s+(?:\w+\s+)?=', block.split('\n)', 1)[0], re.M)
            exists = not names or any(re.search(r'\b' + n + r'\s+(?:\w+\s+)?=', source) for n in names)
            name = names[0] if names else ''
        else:
            name = re.search(r'^(?:type|const|var)\s+(\w+)', line).group(1)
            exists = re.search(r'\b(?:type|const|var)\s+' + name + r'\b', source)
        if exists:
            continue
        # New counters cannot change the tested pre-counter negative-cache body.
        if name == 'evaluateTempUnschedulable':
            block = '''func (s *RateLimitService) evaluateTempUnschedulable(ctx context.Context, account *Account, statusCode int, responseBody []byte, requestedModel ...string) tempUnschedAttempt {
    if s.tryTempUnschedulable(ctx, account, statusCode, responseBody, requestedModel...) { return tempUnschedTriggered }
    return tempUnschedNoMatch
}
'''
        pieces.append(block.rstrip())
    append(path, '\n\n'.join(pieces) + '\n', [])

# The Issue600 network fixture gained an optional per-key configurator. Port it
# into the historical harness, preserving every network assertion and callback.
path = 'backend/internal/service/api_key.go'
field(path, 'APIKey', 'ModelMapping', 'map[string]string')
path = 'backend/internal/service/openai_ws_http_bridge.go'
field(path, 'openAIWSHTTPBridgeToolState', 'ClientTools', 'json.RawMessage')
path = 'backend/internal/handler/openai_ws_v2_passthrough_cyber_test.go'
source = read(path)
if source and 'func newOpenAIWSPassthroughHandlerHarnessWithKey(' not in source:
    signature = re.search(r'^func newOpenAIWSPassthroughHandlerHarnessWithCache\([^\n]+', source, re.M)
    if signature:
        old = signature.group()
        wrapper = old + '\n\treturn newOpenAIWSPassthroughHandlerHarnessWithKey(t, upstreamURL, clientHeaders, extraSettings, gatewayCache, nil, configure...)\n}\n\n'
        actual = old.replace('newOpenAIWSPassthroughHandlerHarnessWithCache(', 'newOpenAIWSPassthroughHandlerHarnessWithKey(').replace('gatewayCache service.GatewayCache, configure ', 'gatewayCache service.GatewayCache, configureKey func(*service.APIKey), configure ')
        source = source[:signature.start()] + wrapper + actual + source[signature.end():]
        anchor = '\tfinalHeaders := make(chan http.Header, 4)'
        if source.count(anchor) != 1:
            raise RuntimeError('historical Issue600 harness anchor changed')
        source = source.replace(anchor, '\tif configureKey != nil { configureKey(apiKey) }\n' + anchor)
        write(path, source)
