"""Exercise deployment with fake Docker; never touch a running service."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / 'action-deploy.sh'
NEW_IMAGE = 'ghcr.io/alanbulan/sub2apipro:sha-' + 'a' * 40
OLD_IMAGE = 'ghcr.io/alanbulan/sub2apipro:sha-' + 'b' * 40

FAKE_TOOL = r'''#!/usr/bin/env python3
import json, os, pathlib, sys
root = pathlib.Path(os.environ['TEST_DEPLOY_ROOT'])
state_file = root / 'state.json'
state = json.loads(state_file.read_text())
tool = pathlib.Path(sys.argv[0]).name
args = sys.argv[1:]
with (root / 'calls.jsonl').open('a') as log:
    log.write(json.dumps([tool, args, os.environ.get('APP_IMAGE')]) + '\n')
mode = os.environ.get('TEST_MODE', '')
old = os.environ['TEST_OLD_IMAGE']
new = os.environ['TEST_NEW_IMAGE']
def save():
    state_file.write_text(json.dumps(state))
if tool == 'realpath':
    print('/opt/sub2api' if args[0] == str(root) else args[0])
elif tool == 'df':
    available = 9000000
    if mode == 'low-before' or (mode == 'low-after' and state.get('pulled')):
        available = 100
    if mode == 'low-docker' and args[-1] == '/fake/docker':
        available = 100
    if mode == 'low-inodes' and args[0] == '-Pi':
        available = 20
    print('Filesystem blocks used available capacity mount')
    print(f'fake 10000000 1000 {available} 1% /')
elif tool == 'sleep':
    pass
elif tool == 'mv':
    if mode == 'persist-failure' and args[0] == '.env.action-next':
        sys.exit(1)
    os.replace(args[0], args[1])
elif tool == 'docker':
    if args[0] == 'inspect':
        fmt = args[2]
        if 'com.docker.compose.project' in fmt:
            print('sub2api')
        elif 'Config.Image' in fmt:
            print(state.get('image', old))
        else:
            print('unhealthy' if mode == 'unhealthy' else 'healthy')
    elif args[0] == 'info':
        print('/fake/docker')
    elif args[0] == 'compose':
        if 'pull' in args:
            if mode == 'pull-failure':
                sys.exit(1)
            state['pulled'] = True
            save()
        if 'up' in args:
            state['image'] = os.environ['APP_IMAGE']
            save()
    elif args[0] == 'stop':
        state['stopped'] = True
        save()
    elif args[0] == 'exec':
        assert args[1] == 'sub2api-postgres'
        assert 'psql' in args[4] and 'pg_dump' not in args[4]
        count = 2 if mode == 'counts-failure' and state.get('image') == new else 1
        print(f'accounts|{count}\napi_keys|1\ngroups|1\nusers|1')
    elif args[:2] == ['image', 'ls']:
        print(new + '\n' + old + '\nredis:8-alpine')
    elif args[:2] == ['image', 'rm']:
        assert args[2] == old
        if mode == 'cleanup-failure':
            sys.exit(1)
    else:
        raise AssertionError(args)
else:
    raise AssertionError(tool)
'''


class DeploymentTests(unittest.TestCase):
    def run_deploy(self, mode=''):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        root = Path(temporary.name)
        bin_dir = root / 'bin'
        bin_dir.mkdir()
        for tool in ('docker', 'df', 'realpath', 'sleep', 'mv'):
            executable = bin_dir / tool
            executable.write_text(FAKE_TOOL)
            executable.chmod(0o755)
        self.original_env = 'SECRET=test-only\nAPP_IMAGE=' + OLD_IMAGE + '\n\n'
        self.original_compose = 'services: original\n\n'
        (root / '.env').write_text(self.original_env)
        (root / '.env').chmod(0o600)
        manifest = root / 'docker-compose.local.yml'
        manifest.write_text(self.original_compose)
        manifest.chmod(0o640)
        staged = root / '.deploy' / ('a' * 40) / manifest.name
        staged.parent.mkdir(parents=True)
        staged.write_text('services: updated\n')
        (root / 'state.json').write_text('{}')
        env = dict(os.environ, PATH=str(bin_dir) + ':' + os.environ['PATH'],
                   TEST_DEPLOY_ROOT=str(root), TEST_MODE=mode,
                   TEST_NEW_IMAGE=NEW_IMAGE, TEST_OLD_IMAGE=OLD_IMAGE)
        result = subprocess.run(['bash', str(SCRIPT), str(root), manifest.name,
                                 str(staged), NEW_IMAGE], env=env,
                                text=True, capture_output=True, timeout=15)
        calls = [json.loads(line) for line in (root / 'calls.jsonl').read_text().splitlines()]
        self.assertFalse((root / 'backups').exists())
        self.assertFalse((root / '.env.action-next').exists())
        self.assertFalse((root / '.compose.action-next').exists())
        self.assertEqual((root / '.env').stat().st_mode & 0o777, 0o600)
        self.assertEqual(manifest.stat().st_mode & 0o777, 0o640)
        return root, result, calls

    def test_success_without_snapshots(self):
        root, result, calls = self.run_deploy()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('APP_IMAGE=' + NEW_IMAGE, (root / '.env').read_text())
        self.assertEqual((root / 'docker-compose.local.yml').read_text(), 'services: updated\n')
        self.assertTrue(any(args[:2] == ['image', 'rm'] for tool, args, image in calls))
        self.assertFalse(any('pg_dump' in str(args) or 'SAVE' in args for tool, args, image in calls))

    def test_disk_or_download_failure_leaves_application_running(self):
        for mode in ('low-before', 'low-after', 'low-docker', 'low-inodes', 'pull-failure'):
            with self.subTest(mode=mode):
                root, result, calls = self.run_deploy(mode)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(args[0] == 'stop' for tool, args, image in calls))
                self.assertEqual((root / '.env').read_text(), self.original_env)
                if mode in ('low-before', 'low-docker', 'low-inodes'):
                    self.assertFalse(any('pull' in args for tool, args, image in calls))

    def test_failure_restarts_previous_image_and_restores_configuration(self):
        for mode in ('unhealthy', 'counts-failure', 'persist-failure'):
            with self.subTest(mode=mode):
                root, result, calls = self.run_deploy(mode)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(json.loads((root / 'state.json').read_text())['image'], OLD_IMAGE)
                self.assertEqual((root / '.env').read_text(), self.original_env)
                self.assertEqual((root / 'docker-compose.local.yml').read_text(), self.original_compose)
                self.assertFalse(any(args[:2] == ['image', 'rm'] for tool, args, image in calls))

    def test_cleanup_failure_does_not_undo_successful_deployment(self):
        root, result, calls = self.run_deploy('cleanup-failure')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(json.loads((root / 'state.json').read_text())['image'], NEW_IMAGE)


if __name__ == '__main__':
    unittest.main()
