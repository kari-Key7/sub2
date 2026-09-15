#!/usr/bin/env bash
# =============================================================================
# 单机 docker compose 的安全升级脚本（也可手动执行）
#
#   upgrade.sh <image-ref> [--no-backup]   拉取新镜像并升级 app 容器
#   upgrade.sh --rollback                  回滚到上一次的镜像（sub2api:prev）
#
# 流程：备份（pg_dump + /app/data 卷）→ 拉取镜像 → 把当前 latest 记为 prev →
#       新镜像打成 latest → 只重建 app 容器（不碰 postgres/redis/数据卷）→
#       健康检查 → 失败自动回滚到 prev 并以非 0 退出。
#
# 切换方式是重新打本地标签 APP_IMAGE（默认 sub2api:latest），因此服务器上的
# docker-compose.yml / .env 无需任何修改。
#
# 环境变量（均有默认值）：
#   DEPLOY_PATH      compose 所在目录（默认：脚本所在目录）
#   APP_SERVICE      compose 里的应用服务名（默认 sub2api）
#   APP_CONTAINER    应用容器名（默认 sub2api）
#   APP_IMAGE        compose 引用的本地镜像名（默认 sub2api:latest）
#   PG_CONTAINER     PostgreSQL 容器名（默认 sub2api-postgres）
#   HEALTH_URL       健康检查地址（默认 http://localhost:8080/health）
#   HEALTH_TIMEOUT   健康检查最长等待秒数（默认 150）
#   HEALTH_INTERVAL  健康检查轮询间隔秒数（默认 3）
#   BACKUP_KEEP      每类备份保留份数（默认 7）
#   GHCR_USER/GHCR_TOKEN  设置后先 docker login ghcr.io，结束时 logout
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_PATH="${DEPLOY_PATH:-${SCRIPT_DIR}}"
APP_SERVICE="${APP_SERVICE:-sub2api}"
APP_CONTAINER="${APP_CONTAINER:-sub2api}"
APP_IMAGE="${APP_IMAGE:-sub2api:latest}"
PREV_IMAGE="${APP_IMAGE%%:*}:prev"
PG_CONTAINER="${PG_CONTAINER:-sub2api-postgres}"
HEALTH_URL="${HEALTH_URL:-http://localhost:8080/health}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-150}"
HEALTH_INTERVAL="${HEALTH_INTERVAL:-3}"
BACKUP_KEEP="${BACKUP_KEEP:-7}"
BACKUP_DIR="${DEPLOY_PATH}/backups"

log() {
    printf '[upgrade] %s\n' "$*"
}

die() {
    printf '[upgrade] ERROR: %s\n' "$*" >&2
    exit 1
}

usage() {
    sed -n '2,12p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
    exit "${1:-0}"
}

compose() {
    if docker compose version >/dev/null 2>&1; then
        docker compose "$@"
    else
        docker-compose "$@"
    fi
}

image_exists() {
    docker image inspect "$1" >/dev/null 2>&1
}

# 每类备份只保留最新的 BACKUP_KEEP 份（文件名含时间戳，按名字排序即按时间排序）。
prune_backups() {
    local pattern=$1
    local files=()
    shopt -s nullglob
    files=( "${BACKUP_DIR}"/${pattern} )
    shopt -u nullglob
    (( ${#files[@]} > BACKUP_KEEP )) || return 0
    printf '%s\n' "${files[@]}" | sort | head -n "$(( ${#files[@]} - BACKUP_KEEP ))" |
        while IFS= read -r old; do rm -f -- "${old}"; done
}

backup() {
    local stamp
    stamp="$(date +%Y%m%d%H%M%S)"
    mkdir -p "${BACKUP_DIR}"

    log "备份数据库 → backups/db_${stamp}.dump"
    # 用户名/库名直接取容器内的 POSTGRES_* 环境变量，无需解析 .env。
    docker exec "${PG_CONTAINER}" sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' \
        > "${BACKUP_DIR}/db_${stamp}.dump"
    [[ -s "${BACKUP_DIR}/db_${stamp}.dump" ]] || die "数据库备份为空，终止升级"

    log "备份 /app/data 卷 → backups/data_${stamp}.tgz"
    # --volumes-from 同时兼容命名卷与目录挂载两种 compose 写法。
    docker run --rm --volumes-from "${APP_CONTAINER}" -v "${BACKUP_DIR}:/backup" alpine \
        tar czf "/backup/data_${stamp}.tgz" -C /app/data .

    prune_backups 'db_*.dump'
    prune_backups 'data_*.tgz'
}

wait_healthy() {
    local waited=0
    while (( waited < HEALTH_TIMEOUT )); do
        if curl -fsS -o /dev/null --max-time 5 "${HEALTH_URL}"; then
            return 0
        fi
        sleep "${HEALTH_INTERVAL}"
        waited=$((waited + HEALTH_INTERVAL))
    done
    return 1
}

restart_app() {
    ( cd "${DEPLOY_PATH}" && compose up -d --no-deps "${APP_SERVICE}" )
}

rollback() {
    image_exists "${PREV_IMAGE}" || die "没有 ${PREV_IMAGE}，无法回滚"
    log "回滚：${PREV_IMAGE} → ${APP_IMAGE}"
    docker tag "${PREV_IMAGE}" "${APP_IMAGE}"
    restart_app
    if wait_healthy; then
        log "回滚完成，服务健康"
    else
        die "回滚后健康检查仍失败，请人工介入"
    fi
}

logout_registry() {
    [[ -n "${GHCR_TOKEN:-}" ]] && docker logout ghcr.io >/dev/null 2>&1 || true
}

# ----------------------------------------------------------------------------
# 参数解析
# ----------------------------------------------------------------------------
IMAGE_REF=""
DO_BACKUP=1
DO_ROLLBACK=0
for arg in "$@"; do
    case "${arg}" in
        --rollback) DO_ROLLBACK=1 ;;
        --no-backup) DO_BACKUP=0 ;;
        -h|--help) usage 0 ;;
        -*) die "未知参数：${arg}" ;;
        *) IMAGE_REF="${arg}" ;;
    esac
done

[[ -d "${DEPLOY_PATH}" ]] || die "DEPLOY_PATH 不存在：${DEPLOY_PATH}"

if (( DO_ROLLBACK )); then
    rollback
    exit 0
fi

[[ -n "${IMAGE_REF}" ]] || usage 1

# ----------------------------------------------------------------------------
# 升级
# ----------------------------------------------------------------------------
trap logout_registry EXIT

if (( DO_BACKUP )); then
    backup
else
    log "已跳过备份（--no-backup）"
fi

if [[ -n "${GHCR_TOKEN:-}" ]]; then
    log "登录 ghcr.io（${GHCR_USER:?GHCR_USER is required with GHCR_TOKEN}）"
    printf '%s' "${GHCR_TOKEN}" | docker login ghcr.io -u "${GHCR_USER}" --password-stdin >/dev/null
fi

log "拉取镜像 ${IMAGE_REF}"
docker pull "${IMAGE_REF}" >/dev/null

if image_exists "${APP_IMAGE}"; then
    log "记录当前镜像为 ${PREV_IMAGE}"
    docker tag "${APP_IMAGE}" "${PREV_IMAGE}"
fi
docker tag "${IMAGE_REF}" "${APP_IMAGE}"

log "重建 ${APP_SERVICE} 容器（不影响 postgres / redis / 数据卷）"
restart_app

log "等待健康检查 ${HEALTH_URL}（最长 ${HEALTH_TIMEOUT}s）"
if ! wait_healthy; then
    log "健康检查失败，最近日志："
    docker logs --tail 100 "${APP_CONTAINER}" 2>&1 | sed 's/^/    /' || true
    if image_exists "${PREV_IMAGE}"; then
        rollback
    fi
    die "升级失败，已回滚到上一版本"
fi

{
    printf 'image=%s\n' "${IMAGE_REF}"
    printf 'deployed_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
} > "${DEPLOY_PATH}/DEPLOYED"

log "升级完成：${IMAGE_REF}"
