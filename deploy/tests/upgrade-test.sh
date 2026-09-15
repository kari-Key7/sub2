#!/bin/bash
# Exercises deploy/upgrade.sh against fake docker/curl binaries:
# backup → pull → tag switch → compose up → health check, plus rollback paths.

set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "${TEST_DIR}/.." && pwd)"
SCRIPT="${DEPLOY_DIR}/upgrade.sh"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-upgrade-test.XXXXXX")"

cleanup() {
    rm -rf "${TEST_ROOT}"
}
trap cleanup EXIT

fail() {
    printf 'FAIL: %s\n' "$*" >&2
    exit 1
}

assert_logged() {
    grep -qF -- "$1" "${FAKE_DOCKER_LOG}" || fail "expected call not found: $1"
}

assert_not_logged() {
    grep -qF -- "$1" "${FAKE_DOCKER_LOG}" && fail "unexpected call found: $1"
    return 0
}

# assert_order A B: A must appear before B in the call log.
assert_order() {
    local first second
    first="$(grep -nF -- "$1" "${FAKE_DOCKER_LOG}" | head -1 | cut -d: -f1)"
    second="$(grep -nF -- "$2" "${FAKE_DOCKER_LOG}" | head -1 | cut -d: -f1)"
    [[ -n "${first}" && -n "${second}" ]] || fail "assert_order: missing call ($1 / $2)"
    [[ "${first}" -lt "${second}" ]] || fail "expected [$1] before [$2]"
}

new_case() {
    local name=$1
    DEPLOY_PATH="${TEST_ROOT}/${name}"
    mkdir -p "${DEPLOY_PATH}"
    touch "${DEPLOY_PATH}/docker-compose.yml"
    FAKE_DOCKER_LOG="${DEPLOY_PATH}/calls.log"
    : > "${FAKE_DOCKER_LOG}"
    export DEPLOY_PATH FAKE_DOCKER_LOG
}

export PATH="${TEST_DIR}/fixtures/upgrade-bin:${PATH}"
export FAKE_IMAGES="sub2api:latest"
export HEALTH_TIMEOUT=2
export HEALTH_INTERVAL=1
unset FAKE_HEALTH FAKE_PULL_FAIL GHCR_TOKEN GHCR_USER || true

REF="ghcr.io/kari-key7/sub2:sha-abc1234"

# ---------------------------------------------------------------------------
# 1. Happy path: backup, login, pull, prev tag, switch, up, health, logout.
# ---------------------------------------------------------------------------
new_case happy
GHCR_USER=bot GHCR_TOKEN=secret "${SCRIPT}" "${REF}" >/dev/null

assert_logged "docker exec sub2api-postgres sh -c pg_dump"
assert_logged "docker run --rm --volumes-from sub2api -v ${DEPLOY_PATH}/backups:/backup alpine tar czf /backup/"
assert_logged "docker login ghcr.io -u bot --password-stdin"
assert_logged "docker pull ${REF}"
assert_logged "docker tag sub2api:latest sub2api:prev"
assert_logged "docker tag ${REF} sub2api:latest"
assert_logged "docker compose up -d --no-deps sub2api"
assert_logged "curl"
assert_logged "docker logout ghcr.io"

assert_order "docker exec sub2api-postgres" "docker pull ${REF}"
assert_order "docker tag sub2api:latest sub2api:prev" "docker tag ${REF} sub2api:latest"
assert_order "docker tag ${REF} sub2api:latest" "docker compose up -d --no-deps sub2api"
assert_order "docker compose up -d --no-deps sub2api" "curl"

ls "${DEPLOY_PATH}"/backups/db_*.dump >/dev/null 2>&1 || fail "database backup file missing"
grep -q 'FAKE-PG-DUMP' "${DEPLOY_PATH}"/backups/db_*.dump || fail "database backup is empty"
ls "${DEPLOY_PATH}"/backups/data_*.tgz >/dev/null 2>&1 || fail "data volume backup file missing"
grep -q "^image=${REF}$" "${DEPLOY_PATH}/DEPLOYED" || fail "DEPLOYED record does not contain the image ref"
# The token must never be visible on a docker command line.
grep -q 'secret' "${FAKE_DOCKER_LOG}" && fail "token leaked onto a command line"

# ---------------------------------------------------------------------------
# 2. Health check fails → roll back to prev and exit non-zero.
# ---------------------------------------------------------------------------
new_case rollback-on-health
if FAKE_HEALTH=fail "${SCRIPT}" "${REF}" >/dev/null 2>&1; then
    fail "upgrade should fail when the health check never passes"
fi
assert_logged "docker tag ${REF} sub2api:latest"
assert_logged "docker tag sub2api:prev sub2api:latest"
assert_order "docker tag ${REF} sub2api:latest" "docker tag sub2api:prev sub2api:latest"
assert_logged "docker logs --tail 100 sub2api"
[[ "$(grep -c 'docker compose up -d --no-deps sub2api' "${FAKE_DOCKER_LOG}")" -eq 2 ]] || fail "expected a second compose up for the rollback"
if [[ -e "${DEPLOY_PATH}/DEPLOYED" ]] && grep -q "^image=${REF}$" "${DEPLOY_PATH}/DEPLOYED"; then
    fail "failed deploy must not be recorded as deployed"
fi

# ---------------------------------------------------------------------------
# 3. Explicit rollback: no backup, no pull, just retag prev and restart.
# ---------------------------------------------------------------------------
new_case rollback-cmd
FAKE_IMAGES="sub2api:latest sub2api:prev" "${SCRIPT}" --rollback >/dev/null
assert_logged "docker tag sub2api:prev sub2api:latest"
assert_logged "docker compose up -d --no-deps sub2api"
assert_not_logged "docker pull"
assert_not_logged "docker exec sub2api-postgres"
assert_not_logged "docker tag sub2api:latest sub2api:prev"

# Rollback without a prev image must refuse instead of restarting blindly.
new_case rollback-no-prev
if "${SCRIPT}" --rollback >/dev/null 2>&1; then
    fail "rollback should fail when sub2api:prev does not exist"
fi
assert_not_logged "docker compose up"

# ---------------------------------------------------------------------------
# 4. --no-backup skips both backups; no token → no login/logout.
# ---------------------------------------------------------------------------
new_case no-backup
"${SCRIPT}" --no-backup "${REF}" >/dev/null
assert_not_logged "docker exec sub2api-postgres"
assert_not_logged "docker run --rm --volumes-from"
assert_not_logged "docker login"
assert_not_logged "docker logout"
assert_logged "docker pull ${REF}"

# ---------------------------------------------------------------------------
# 5. First deploy (no sub2api:latest yet) must not try to tag a prev image.
# ---------------------------------------------------------------------------
new_case first-deploy
FAKE_IMAGES="" "${SCRIPT}" --no-backup "${REF}" >/dev/null
assert_not_logged "docker tag sub2api:latest sub2api:prev"
assert_logged "docker tag ${REF} sub2api:latest"

# ---------------------------------------------------------------------------
# 6. Backup retention keeps only the newest BACKUP_KEEP of each kind.
# ---------------------------------------------------------------------------
new_case retention
mkdir -p "${DEPLOY_PATH}/backups"
for i in $(seq 1 9); do
    printf 'old\n' > "${DEPLOY_PATH}/backups/db_2000010100000${i}.dump"
    printf 'old\n' > "${DEPLOY_PATH}/backups/data_2000010100000${i}.tgz"
done
BACKUP_KEEP=3 "${SCRIPT}" "${REF}" >/dev/null
[[ "$(ls "${DEPLOY_PATH}"/backups/db_*.dump | wc -l | tr -d ' ')" -eq 3 ]] || fail "expected 3 db backups after pruning"
[[ "$(ls "${DEPLOY_PATH}"/backups/data_*.tgz | wc -l | tr -d ' ')" -eq 3 ]] || fail "expected 3 data backups after pruning"
[[ ! -e "${DEPLOY_PATH}/backups/db_20000101000001.dump" ]] || fail "oldest db backup should have been pruned"

# ---------------------------------------------------------------------------
# 7. Pull failure aborts before touching tags or containers.
# ---------------------------------------------------------------------------
new_case pull-fail
if FAKE_PULL_FAIL=1 "${SCRIPT}" --no-backup "${REF}" >/dev/null 2>&1; then
    fail "upgrade should fail when the image cannot be pulled"
fi
assert_not_logged "docker tag"
assert_not_logged "docker compose up"

printf 'upgrade-test: all cases passed\n'
