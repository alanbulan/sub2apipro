#!/usr/bin/env python3
"""Make remote test failures visible in public Actions annotations."""
from pathlib import Path
import re
import sys
for filename in sys.argv[1:]:
    path = Path(filename)
    if not path.exists():
        continue
    lines = path.read_text(errors='replace').splitlines()
    reported = 0
    for index, line in enumerate(lines):
        if re.search(r"--- FAIL:|^FAIL\s|^not ok|^FAIL:|Error:|error TS[0-9]", line):
            message = '\n'.join(lines[max(0,index-18):index+22])[:10000]
            print('::error::' + message.replace('%', '%25').replace('\r', '%0D').replace('\n', '%0A'))
            reported += 1
            if reported >= 15:
                break
    if not reported:
        print('No failure markers in ' + path.name)
