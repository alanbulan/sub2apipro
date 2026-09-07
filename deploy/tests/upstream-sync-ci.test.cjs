// Run in GitHub Actions only. Network and Git fixtures never touch production.
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const http = require('node:http');
const { spawn, spawnSync } = require('node:child_process');
const { waitForCI, checkpoint, writeJSON } = require('../cron/upstream-sync-ci.cjs');

const sha = 'a'.repeat(40);
const branch = 'sync/upstream-candidate';
const options = { repository: 'fixture/repo', branch, sha, timeoutMs: 600000, pollMs: 60000 };
const run = (conclusion, overrides = {}) => ({ id: 12, event: 'push', head_branch: branch, head_sha: sha,
  status: conclusion ? 'completed' : 'in_progress', conclusion, html_url: 'https://github.com/fixture/repo/actions/runs/12', ...overrides });
const response = (runs, status = 200, headers = {}) => new Response(JSON.stringify({ workflow_runs: runs }), { status, headers });

async function simulate(replies, overrides = {}) {
  let time = 1000000;
  const requests = [], logs = [], states = [], sleeps = [];
  const code = await waitForCI({ ...options, ...overrides }, {
    now: () => time,
    sleep: async ms => { sleeps.push(ms); time += ms; },
    log: value => logs.push(value), save: value => states.push(value),
    fetch: async (url, init) => {
      requests.push({ url, init, time });
      const value = replies.length > 1 ? replies.shift() : replies[0];
      if (value instanceof Error) throw value;
      return typeof value === 'function' ? value() : value;
    },
  });
  return { code, requests, logs, states, sleeps };
}

test('honors rate-limit reset, keeps running URL when conclusion is null, then passes', async () => {
  const result = await simulate([
    response([], 403, { 'x-ratelimit-remaining': '0', 'x-ratelimit-reset': '1180' }),
    response([run(null)]), response([run('success')]),
  ]);
  assert.equal(result.code, 0);
  assert.equal(result.requests[1].time, 1181000);
  assert.ok(result.sleeps.every(ms => ms <= 60000));
  assert.ok(result.logs.some(line => line.includes('in_progress') && line.includes('/actions/runs/12')));
  assert.equal(result.states[0].state, 'unavailable');
  assert.equal(result.states.at(-1).state, 'passed');
  assert.equal(result.requests[0].url.searchParams.get('head_sha'), sha);
  assert.equal(result.requests[0].url.searchParams.get('branch'), branch);
});

test('retries network, server and malformed JSON responses without treating them as failed CI', async () => {
  const result = await simulate([new Error('network offline'), response([], 503), new Response('invalid JSON'), response([run('success')])]);
  assert.equal(result.code, 0);
  assert.ok(result.states.slice(0, 3).every(state => state.state === 'unavailable'));
});

test('timeout stays pending and honors Retry-After even past this invocation deadline', async () => {
  const result = await simulate([() => response([], 429, { 'retry-after': '3600' })], { timeoutMs: 120000 });
  assert.equal(result.code, 2);
  assert.equal(result.requests.length, 1);
  assert.equal(result.states.at(-1).state, 'pending');
});

test('success for another branch or SHA cannot authorize promotion', async () => {
  const result = await simulate([() => response([run('success', { head_branch: 'main' }), run('success', { head_sha: 'b'.repeat(40) })])], { timeoutMs: 60000 });
  assert.equal(result.code, 2);
  assert.ok(result.states.every(state => state.state === 'pending'));
});

test('explicit failure, cancellation and runner timeout are non-passing CI results', async () => {
  for (const conclusion of ['failure', 'cancelled', 'timed_out']) {
    const result = await simulate([response([run(conclusion)])]);
    assert.equal(result.code, 1);
    assert.equal(result.states.at(-1).conclusion, conclusion);
  }
});

test('missing permissions are an unavailable query; tokens are never logged or persisted', async () => {
  const result = await simulate([response([], 401)], { token: 'private-fixture-token' });
  assert.equal(result.code, 3);
  assert.equal(result.states.at(-1).state, 'unavailable');
  assert.equal(result.requests[0].init.headers.Authorization, 'Bearer private-fixture-token');
  assert.ok(!JSON.stringify([result.logs, result.states]).includes('private-fixture-token'));
});

test('rejects invalid checkpoint references and a production candidate branch', () => {
  const good = { before_head: sha, candidate_head: 'b'.repeat(40), remote_head: sha, branch };
  assert.deepEqual(checkpoint(good), good);
  assert.throws(() => checkpoint({ ...good, candidate_head: 'HEAD' }));
  assert.throws(() => checkpoint({ ...good, branch: 'main' }));
  assert.throws(() => checkpoint({ ...good, branch: 'sync/branch $(command)' }));
});

function git(cwd, args, env) {
  const result = spawnSync('git', args, { cwd, env: env || process.env, encoding: 'utf8' });
  assert.equal(result.status, 0, result.stderr);
  return result.stdout.trim();
}

async function fixture(t) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'upstream-sync-ci-'));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  const repo = path.join(directory, 'repo');
  const remote = path.join(directory, 'remote.git');
  fs.mkdirSync(repo);
  git(repo, ['init', '-q', '-b', 'main']);
  git(repo, ['config', 'user.email', 'fixture@example.invalid']);
  git(repo, ['config', 'user.name', 'CI fixture']);
  for (const name of ['upstream-sync', 'upstream-sync-ci.cjs']) {
    fs.mkdirSync(path.join(repo, 'deploy/cron'), { recursive: true });
    fs.copyFileSync(path.join(__dirname, '../cron', name), path.join(repo, 'deploy/cron', name));
  }
  fs.mkdirSync(path.join(repo, 'custom'));
  fs.writeFileSync(path.join(repo, 'custom/protected-paths.txt'), 'custom/\ndeploy/cron/\n');
  fs.writeFileSync(path.join(repo, '.gitignore'), '.codex-upstream-sync/\n');
  git(repo, ['add', '.']);
  git(repo, ['commit', '-qm', 'base']);
  const before = git(repo, ['rev-parse', 'HEAD']);
  git(repo, ['clone', '--bare', '-q', repo, remote]);
  fs.writeFileSync(path.join(repo, 'feature.txt'), 'upstream change\n');
  git(repo, ['add', 'feature.txt']);
  git(repo, ['commit', '-qm', 'candidate']);
  const candidate = git(repo, ['rev-parse', 'HEAD']);
  git(repo, ['push', '-q', remote, `HEAD:refs/heads/${branch}`]);
  // Only this disposable fixture is reset to emulate a previous sync exit.
  git(repo, ['reset', '--hard', before]);
  git(repo, ['remote', 'add', 'origin', 'git@github.com:fixture/repo.git']);
  git(repo, ['remote', 'add', 'upstream', remote]);
  git(repo, ['remote', 'set-url', '--push', 'upstream', 'DISABLED']);
  const ssh = path.join(directory, 'ssh.cjs');
  fs.writeFileSync(ssh, `const {spawnSync}=require('node:child_process');
const command=process.argv.at(-1);const match=/^git-(upload|receive)-pack /.exec(command);
if(match){const r=spawnSync('git-'+match[1]+'-pack',[process.env.SYNC_TEST_REMOTE],{stdio:'inherit'});process.exit(r.status ?? 1);}
process.exit(0);\n`);
  const stateDir = path.join(repo, '.codex-upstream-sync');
  fs.mkdirSync(stateDir);
  const checkpointFile = path.join(stateDir, 'candidate.json');
  writeJSON(checkpointFile, { before_head: before, candidate_head: candidate, remote_head: candidate, branch });
  fs.writeFileSync(path.join(stateDir, 'last-seen-head'), `${before}\n`);
  const state = { conclusion: 'success', status: 200, requests: 0 };
  const server = http.createServer((req, res) => {
    state.requests++;
    res.writeHead(state.status, { 'content-type': 'application/json', 'retry-after': '60' });
    res.end(JSON.stringify({ workflow_runs: [run(state.conclusion, { head_sha: candidate })] }));
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  t.after(() => { server.closeAllConnections(); server.close(); });
  const quote = value => "'" + value.replace(/'/g, "'\\''") + "'";
  const env = { ...process.env, REPO_DIR: repo, GITHUB_API_URL: `http://127.0.0.1:${server.address().port}`,
    GITHUB_TOKEN: '', GH_TOKEN: '', GITHUB_TOKEN_FILE: '', SYNC_TEST_REMOTE: remote,
    GIT_SSH_COMMAND: `${quote(process.execPath)} ${quote(ssh)}`, CANDIDATE_CI_TIMEOUT_SECONDS: '1', CANDIDATE_CI_POLL_SECONDS: '1' };
  const invoke = () => new Promise((resolve, reject) => {
    const child = spawn('/bin/sh', ['deploy/cron/upstream-sync'], { cwd: repo, env, stdio: 'ignore' });
    child.on('error', reject);
    child.on('exit', code => {
      const logs = path.join(stateDir, 'logs');
      state.log = fs.existsSync(logs) ? fs.readdirSync(logs).map(name => fs.readFileSync(path.join(logs, name), 'utf8')).join('\n') : 'No sync log created';
      resolve(code);
    });
  });
  return { repo, remote, stateDir, checkpointFile, before, candidate, state, invoke };
}

test('wrapper resumes an unknown CI result and promotes the same saved candidate on retry', async t => {
  const f = await fixture(t);
  f.state.status = 403;
  assert.notEqual(await f.invoke(), 0);
  assert.ok(fs.existsSync(f.checkpointFile));
  assert.equal(git(f.remote, ['rev-parse', 'main']), f.before);
  assert.equal(fs.readFileSync(path.join(f.stateDir, 'last-seen-head'), 'utf8').trim(), f.before);
  f.state.status = 200;
  assert.equal(await f.invoke(), 0, f.state.log);
  assert.equal(git(f.remote, ['rev-parse', 'main']), f.candidate);
  assert.equal(git(f.repo, ['rev-parse', 'HEAD']), f.candidate);
  assert.equal(git(f.remote, ['rev-parse', branch]), f.candidate);
  assert.equal(fs.readFileSync(path.join(f.stateDir, 'last-seen-head'), 'utf8').trim(), f.candidate);
  assert.ok(!fs.existsSync(f.checkpointFile));
});

test('wrapper archives an explicit failed CI without promoting or advancing state', async t => {
  const f = await fixture(t);
  f.state.conclusion = 'failure';
  assert.notEqual(await f.invoke(), 0);
  assert.ok(fs.existsSync(path.join(f.stateDir, 'candidate.failed.json')), f.state.log);
  assert.equal(git(f.remote, ['rev-parse', 'main']), f.before);
  assert.equal(fs.readFileSync(path.join(f.stateDir, 'last-seen-head'), 'utf8').trim(), f.before);
});

test('wrapper refuses a replaced remote candidate', async t => {
  const f = await fixture(t);
  git(f.remote, ['update-ref', `refs/heads/${branch}`, f.before]);
  assert.notEqual(await f.invoke(), 0);
  assert.equal(f.state.requests, 0);
  assert.equal(git(f.remote, ['rev-parse', 'main']), f.before);
  assert.ok(fs.existsSync(path.join(f.stateDir, 'candidate.superseded.json')));
});

test('wrapper repairs state after a crash between remote promotion and state write', async t => {
  const f = await fixture(t);
  git(f.remote, ['update-ref', 'refs/heads/main', f.candidate]);
  assert.equal(await f.invoke(), 0, f.state.log);
  assert.equal(fs.readFileSync(path.join(f.stateDir, 'last-seen-head'), 'utf8').trim(), f.candidate);
  assert.ok(!fs.existsSync(f.checkpointFile));
});
