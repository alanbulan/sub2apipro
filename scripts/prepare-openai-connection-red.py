"""Carry the current historical-grant fixture into pre-helper RED trees on CI."""
from pathlib import Path
import re
import sys

baseline, candidate = map(Path, sys.argv[1:])
source = (candidate / "backend/internal/repository/openai_auth_connections_http_integration_test.go").read_text()
start = source.index("func (f *connectionHTTPFixture) seedHistoricalAPIKey(")
end = source.index("\nfunc ", start + 1)
helper = source[start:end]
# These historical sources predate the refresh-intent field. Keep their
# vulnerable production implementation and the candidate's assertions intact.
model = (baseline / "backend/internal/service/openai_auth_connections.go").read_text()
if not re.search(r"\bRefreshState\s+string", model):
    helper = helper.replace('RefreshState: "idle", ', '')
assert "seedHistoricalAPIKey" in helper
imports = '''//go:build integration

package repository_test

import (
    "context"
    "encoding/json"
    "strconv"
    "testing"

    "github.com/Wei-Shaw/sub2api/internal/service"
    "github.com/google/uuid"
    "github.com/stretchr/testify/require"
)

'''
(baseline / "backend/internal/repository/connection_red_fixture_compat_test.go").write_text(imports + helper + "\n")
print("Added historical API-key fixture compatibility; production sources and regression assertions unchanged")
