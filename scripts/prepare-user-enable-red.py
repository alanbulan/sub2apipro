"""Add the independent VIP credit dependency to the historical moderation fixture."""
from pathlib import Path
import sys
p = Path(sys.argv[1])
s = p.read_text()
marker = "type UserHandler struct {"
assert s.count(marker) == 1
assert "autoCredit" not in s
p.write_text(s.replace(marker, marker + "\n\tautoCredit service.AutoCreditRepository"))
