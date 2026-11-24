#!/usr/bin/env bash
# One-time host setup; rerun after changing action-deploy.sh.
set -euo pipefail
test "$(id -u)" = 0
script_dir=$(cd -- "$(dirname -- "$0")" && pwd)
id sub2api-deploy >/dev/null
install -d -m 0700 -o sub2api-deploy -g sub2api-deploy /opt/sub2api/.deploy
install -m 0755 -o root -g root "$script_dir/action-deploy.sh" /usr/local/sbin/sub2api-action-deploy
sudo_rule=$(mktemp)
trap 'rm -f "$sudo_rule"' EXIT
printf '%s\n' 'sub2api-deploy ALL=(root) NOPASSWD: /usr/local/sbin/sub2api-action-deploy' > "$sudo_rule"
visudo -cf "$sudo_rule"
install -m 0440 -o root -g root "$sudo_rule" /etc/sudoers.d/sub2api-action-deploy
