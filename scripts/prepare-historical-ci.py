"""Keep historical RED defects while supplying independent, later interfaces.

Runner only. The original behavioral assertions and source hash checks remain
in the workflow; this helper never changes their expected results.
"""
from pathlib import Path
import re
import subprocess
import sys

mode = sys.argv[1]
root = Path(sys.argv[2] if len(sys.argv) > 2 else '.').resolve()


def current(path):
    return subprocess.check_output(['git', '-C', str(root), 'show', 'HEAD:' + path], text=True)


if mode == 'rate-limit':
    path = 'backend/internal/service/ratelimit_service.go'
    p = root / path
    source = p.read_text()
    pattern = r'type RateLimitService struct \{\n(.*?)\n\}'
    old = re.search(pattern, source, re.S)
    new = re.search(pattern, current(path), re.S)
    assert old and new
    # Only copy missing field declarations; retain every historical method.
    fields = re.findall(r'^\s*(\w+)\s+([^\n/]+?)\s*$', new[1], re.M)
    missing = [(name, kind) for name, kind in fields if not re.search(r'^\s*' + re.escape(name) + r'\s', old[1], re.M)]
    additions = ''.join('\t' + name + ' ' + kind + '\n' for name, kind in missing)
    source = source[:old.start(1)] + additions + source[old.start(1):]
    if any('sync.' in kind for _, kind in missing) and '"sync"' not in source:
        source = source.replace('import (\n', 'import (\n\t"sync"\n', 1)
    p.write_text(source)
elif mode == 'scheduler-callers':
    # Keep the tested scheduling file byte-identical to the pinned source.
    p = root / 'backend/internal/service/openai_account_scheduler.go'
    source = p.read_text()
    old = ', useUpstreamTokenCost, acceptLegacyAccount)'
    assert source.count(old) == 2
    p.write_text(source.replace(old, ', useUpstreamTokenCost)'))
elif mode == 'header-timeout':
    # The old cache is intentionally stale. Image timeout selection was added
    # independently later and is needed to compile current transport callers.
    p = root / 'backend/internal/service/setting_openai_response_header_timeout.go'
    source = p.read_text()
    if 'func (s *SettingService) OpenAIImageResponseHeaderTimeout(' not in source:
        marker = 'type OpenAIResponseHeaderTimeoutSettings struct {\n'
        assert source.count(marker) == 1
        source = source.replace(marker, marker + '\tImageGroupIDs []int64 `json:"image_group_ids"`\n\tImageSeconds int `json:"image_seconds"`\n', 1)
        source += '''
func (s *SettingService) OpenAIImageResponseHeaderTimeout(ctx context.Context, groupID *int64) time.Duration {
    if groupID == nil { return 0 }
    settings, err := s.GetOpenAIResponseHeaderTimeoutSettings(ctx)
    if err != nil || settings == nil || !settings.Enabled || settings.ImageSeconds <= settings.Seconds { return 0 }
    for _, id := range settings.ImageGroupIDs {
        if id == *groupID { return time.Duration(settings.ImageSeconds) * time.Second }
    }
    return 0
}
'''
        p.write_text(source)
else:
    raise SystemExit('unknown historical fixture: ' + mode)
