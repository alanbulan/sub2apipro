// GitHub is the test authority. This helper only reads CI status and saves it.
const fs = require('node:fs');

function writeJSON(file, value) {
  const temporary = `${file}.${process.pid}.tmp`;
  fs.writeFileSync(temporary, `${JSON.stringify(value, null, 2)}\n`, { mode: 0o600 });
  fs.renameSync(temporary, file);
}

function checkpoint(value) {
  for (const key of ['before_head', 'candidate_head', 'remote_head']) {
    if (!/^[0-9a-f]{40}$/.test(value[key] || '')) throw new Error('Invalid candidate checkpoint');
  }
  if (typeof value.branch !== 'string' || !/^[A-Za-z0-9][A-Za-z0-9._/-]*$/.test(value.branch) || value.branch === 'main') {
    throw new Error('Invalid candidate branch in checkpoint');
  }
  return value;
}

function retryDelay(headers, now, fallback) {
  const retryAfter = headers.get('retry-after');
  const retryMs = retryAfter == null ? 0 : /^\d+$/.test(retryAfter)
    ? Number(retryAfter) * 1000 : Date.parse(retryAfter) - now;
  const resetMs = headers.get('x-ratelimit-remaining') === '0'
    ? Number(headers.get('x-ratelimit-reset')) * 1000 - now + 1000 : 0;
  return Math.max(fallback, Number.isFinite(retryMs) ? retryMs : 0, Number.isFinite(resetMs) ? resetMs : 0);
}

async function waitForCI(options, dependencies = {}) {
  const { repository, branch, sha, timeoutMs, pollMs, token = '', apiURL = 'https://api.github.com' } = options;
  if (!/^[\w.-]+\/[\w.-]+$/.test(repository) || !/^[0-9a-f]{40}$/.test(sha)
      || !(timeoutMs > 0) || !(pollMs > 0)) throw new Error('Invalid CI polling configuration');
  const request = dependencies.fetch || globalThis.fetch;
  const now = dependencies.now || Date.now;
  const sleep = dependencies.sleep || (ms => new Promise(resolve => setTimeout(resolve, ms)));
  const save = dependencies.save || (() => {});
  const log = dependencies.log || (message => console.log(`${new Date(now()).toISOString()} ${message}`));
  const deadline = now() + timeoutMs;
  const url = new URL(`${apiURL.replace(/\/$/, '')}/repos/${repository}/actions/workflows/ci.yml/runs`);
  url.search = new URLSearchParams({ event: 'push', branch, head_sha: sha, per_page: '10' }).toString();
  const headers = { Accept: 'application/vnd.github+json', 'User-Agent': 'sub2api-upstream-sync' };
  if (token) headers.Authorization = `Bearer ${token}`;
  let failures = 0;
  let lastRun = {};
  const record = (state, reason) => save({ sha, branch, ...lastRun, state, reason, updated_at: new Date(now()).toISOString() });

  while (now() < deadline) {
    let delay = pollMs;
    try {
      const response = await request(url, { headers, signal: AbortSignal.timeout(Math.max(1, Math.min(30000, deadline - now()))) });
      if (!response.ok) {
        await response.body?.cancel();
        failures += 1;
        delay = retryDelay(response.headers, now(), Math.min(pollMs * 2 ** Math.min(failures - 1, 5), 300000));
        record('unavailable', `GitHub API HTTP ${response.status}; CI result remains unknown`);
        log(`CI status unavailable (HTTP ${response.status}); retry in ${Math.ceil(delay / 1000)}s; candidate ${sha} retained`);
        // Permission/configuration errors need correction, but are not failed tests.
        if (response.status === 401 || response.status === 404) return 3;
      } else {
        const body = await response.json();
        if (!Array.isArray(body.workflow_runs)) throw new Error('Invalid GitHub response');
        failures = 0;
        const run = body.workflow_runs.filter(item => item.event === 'push' && item.head_sha === sha && item.head_branch === branch)
          .sort((a, b) => b.id - a.id)[0];
        if (!run) {
          record('pending', 'No CI run found for this candidate yet');
          log(`Waiting for CI to start for candidate ${sha}`);
        } else {
          lastRun = { run_id: run.id, run_url: run.html_url, conclusion: run.conclusion || null };
          if (run.status === 'completed' && run.conclusion) {
            const passed = run.conclusion === 'success';
            record(passed ? 'passed' : 'failed', `CI completed: ${run.conclusion}`);
            log(`Candidate CI ${run.conclusion}: ${run.html_url}`);
            return passed ? 0 : 1;
          }
          record('running', `CI status: ${run.status}`);
          log(`Waiting for candidate CI (${run.status}): ${run.html_url}`);
        }
      }
    } catch {
      failures += 1;
      delay = Math.min(pollMs * 2 ** Math.min(failures - 1, 5), 300000);
      record('unavailable', 'GitHub request failed or returned invalid JSON; CI result remains unknown');
      log(`CI query unavailable; retry in ${Math.ceil(delay / 1000)}s; candidate ${sha} retained`);
    }
    // Keep individual waits bounded while honoring Retry-After/rate-limit reset.
    const retryAt = Math.min(now() + delay, deadline);
    while (now() < retryAt) await sleep(Math.min(60000, retryAt - now()));
  }
  record('pending', 'CI wait deadline reached; resume this candidate on the next invocation');
  log(`CI wait expired; candidate ${sha} remains pending and can be resumed`);
  return 2;
}

async function main(args) {
  const [command, ...values] = args;
  if (command === 'checkpoint-write') {
    const [file, before_head, candidate_head, remote_head, branch] = values;
    writeJSON(file, checkpoint({ before_head, candidate_head, remote_head, branch }));
    return 0;
  }
  if (command === 'checkpoint-read') {
    const value = checkpoint(JSON.parse(fs.readFileSync(values[0], 'utf8')));
    console.log([value.before_head, value.candidate_head, value.remote_head, value.branch].join(' '));
    return 0;
  }
  if (command !== 'wait') throw new Error('Unknown command');
  const [repository, branch, sha, statusFile, timeout, poll] = values;
  const token = process.env.GITHUB_TOKEN || process.env.GH_TOKEN
    || (process.env.GITHUB_TOKEN_FILE ? fs.readFileSync(process.env.GITHUB_TOKEN_FILE, 'utf8').trim() : '');
  return waitForCI({ repository, branch, sha, timeoutMs: Number(timeout) * 1000,
    pollMs: Number(poll) * 1000, token, apiURL: process.env.GITHUB_API_URL },
  { save: value => writeJSON(statusFile, value) });
}

module.exports = { waitForCI, retryDelay, checkpoint, writeJSON };
if (require.main === module) {
  main(process.argv.slice(2)).then(code => { process.exitCode = code; }).catch(() => {
    console.error('Unable to read CI configuration or save sync state; candidate remains pending. Check token-file access and state-directory permissions.');
    process.exitCode = 3;
  });
}
