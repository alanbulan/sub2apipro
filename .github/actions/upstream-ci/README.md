# Upstream CI checks

These local composite actions carry the steps appended to `backend-ci.yml` by
the 2026-11-16 upstream sync. GitHub limits workflow files to 500 KB, so the
upstream jobs keep their runners, dependencies, matrices, services and
environment in the workflow and call these actions after checking out the
candidate. The existing protected workflow lines remain unchanged.

Each directory is the first 16 hexadecimal characters of SHA-256 of its job ID.
The action's `name` records that job ID. Commands and regression assertions are
preserved; matrix values used inside a composite action are passed explicitly
as inputs. Run steps retain the job's working directory and shell settings.

Historical-source preparation runs only in disposable checkouts on GitHub
runners. `scripts/prepare-sync-ci.py` supplies independent interfaces required by
newer callers while retaining the historical behavior that RED checks exercise.
It does not change regression assertions or report a test result.

Builds, tests, dependency installation and code generation run in GitHub Actions.
The production sync host performs text and syntax checks only.
