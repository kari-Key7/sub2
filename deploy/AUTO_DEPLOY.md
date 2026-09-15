# 自动部署（GitHub Actions → GHCR → 服务器）

`main` 分支上的 CI 全部通过后，[`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml) 会自动：

1. 精确检出 CI 验证过的那个提交，用仓库的 `Dockerfile` 构建 `linux/amd64` 镜像；
2. 推到 GHCR 私有包 `ghcr.io/<owner>/<repo>:sha-<短SHA>` 和 `:latest`；
3. SSH 到服务器，把 [`deploy/upgrade.sh`](upgrade.sh) 传过去并执行：
   备份 → 拉取镜像 → 把当前镜像记为 `sub2api:prev` → 新镜像打成 compose 引用的本地镜像名 →
   `docker compose up -d --no-deps sub2api` → 轮询 `/health` → 失败自动回滚到 `prev`。

只重建应用容器，**不碰 postgres / redis 容器和数据卷**，服务器上的 compose 文件 / `.env`
无需修改（compose 仍然引用本地镜像名，脚本只是把新镜像重新打成这个名字）。

当前线上布局（workflow 里的默认值，2026-09 核对）：compose 目录 `/home/ubuntu/sub2api/deploy`，
compose 文件 `docker-compose.prod.yml`（服务器本地维护，不在仓库里，同一个文件里还有 Caddy），
app 服务用 `build:` 构建，镜像名是 compose 自动生成的 `deploy-sub2api`。脚本把 GHCR 拉下来的镜像
重新打成这个名字再 `up -d --no-deps --force-recreate sub2api`，所以 `build:` 不会被触发。
布局变了就在仓库 Settings → Variables 里设置 `DEPLOY_COMPOSE_FILE` / `DEPLOY_APP_IMAGE` 覆盖。

健康检查默认在应用容器内执行 `wget http://localhost:8080/health`（和 compose 自带 healthcheck 一样），
不要求 8080 发布到宿主机；想改成宿主机探测就设 `HEALTH_URL`。

## 一次性准备

### 服务器

直接用平时运维的 `ubuntu` 用户即可（能用 docker 的用户等同 root，单独建部署用户没有额外收益）。
**不要**对 compose 目录 `chown -R`：`postgres_data/` 是目录挂载，改了属主 PostgreSQL 会起不来。

```bash
# 1. 确认 ubuntu 不用 sudo 就能跑 docker（自动部署时没有 sudo）；报权限错误则执行
#    sudo usermod -aG docker ubuntu 后重新登录
docker ps --format '{{.Names}}'

# 2. 生成只用于部署的 SSH 密钥（在本机执行），公钥加到服务器
ssh-keygen -t ed25519 -C "github-deploy" -f ~/.ssh/sub2api_deploy -N ""
ssh-copy-id -i ~/.ssh/sub2api_deploy.pub ubuntu@<服务器>

# 3. 取服务器主机指纹（写进 DEPLOY_KNOWN_HOSTS）
ssh-keyscan -p 22 <服务器> 2>/dev/null
```

服务器需要 `docker`（含 `compose` 插件或 `docker-compose`）、`curl`；`upgrade.sh` 会用
`docker exec sub2api-postgres pg_dump` 和 `docker run --volumes-from sub2api alpine tar` 做备份，
所以容器名需要是 compose 文件里的 `sub2api` / `sub2api-postgres`（不同的话改 workflow 传入
`APP_CONTAINER` / `PG_CONTAINER`）。

### GitHub 仓库 Secrets（Settings → Secrets and variables → Actions）

| Secret | 内容 |
|---|---|
| `DEPLOY_HOST` | 服务器地址 |
| `DEPLOY_PORT` | SSH 端口，默认 22（可不设） |
| `DEPLOY_USER` | `ubuntu` |
| `DEPLOY_SSH_KEY` | `~/.ssh/sub2api_deploy` 私钥全文 |
| `DEPLOY_KNOWN_HOSTS` | `ssh-keyscan` 输出（可多行） |
| `DEPLOY_PATH` | `/home/ubuntu/sub2api/deploy`（compose 文件所在目录） |

拉取镜像不需要额外密钥：workflow 把本次任务的临时 `GITHUB_TOKEN` 通过 stdin 交给服务器
`docker login ghcr.io`，脚本结束时 `docker logout`，服务器上不保存长期凭据。

### GHCR 包可见性

第一次成功推送后到 GitHub → Packages → `<repo>` → Package settings 确认 **Visibility = Private**。

## 触发方式

- **自动**：把 `dev` 合并到 `main` 并 push；名为 `CI` 的工作流全部通过后 Deploy 自动开始。
  CI 红了不会部署。注意 `workflow_run` 只认默认分支（`main`）上的 `deploy.yml`，
  所以这个文件本身要先合到 `main`。
- **手动**：Actions → Deploy → Run workflow（跳过 CI 门禁；可勾选跳过备份）。
- 同一时刻只跑一个部署，后来的排队等待。
- 想加人工审批：Settings → Environments → `production` → Required reviewers。

## 回滚

自动：健康检查在 `HEALTH_TIMEOUT`（默认 150s）内没通过，脚本自动切回 `sub2api:prev` 并让工作流标红。

手动（在服务器上）：

```bash
cd /home/ubuntu/sub2api/deploy && COMPOSE_FILE=docker-compose.prod.yml APP_IMAGE=deploy-sub2api ./upgrade.sh --rollback
```

回到更早的版本：`docker images ghcr.io/<owner>/<repo>` 里挑一个 `sha-xxxxxxx`，然后

```bash
cd /home/ubuntu/sub2api/deploy && COMPOSE_FILE=docker-compose.prod.yml APP_IMAGE=deploy-sub2api ./upgrade.sh --no-backup ghcr.io/<owner>/<repo>:sha-xxxxxxx
```

数据库迁移是增量且向前兼容的（只新增不删除），回滚二进制不需要回滚数据库。

## 备份

每次升级前在 `<DEPLOY_PATH>/backups/` 生成 `db_<时间>.dump`（`pg_dump -Fc`）和
`data_<时间>.tgz`（`/app/data` 卷：`config.yaml`、安装锁等），每类保留最近 `BACKUP_KEEP`（默认 7）份。

恢复数据库示例：

```bash
docker exec -i sub2api-postgres sh -c 'pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists' < backups/db_<时间>.dump
```

## 本地验证

```bash
/bin/bash deploy/tests/upgrade-test.sh
```

用假的 `docker` / `curl` 跑通升级、健康检查失败回滚、`--rollback`、`--no-backup`、首次部署、备份保留等分支。
