#!/usr/bin/env bash
# Ubuntu 单机部署；全部命令由此入口串行执行。
set -Eeuo pipefail
export LC_ALL=C
umask 077
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
ENV_FILE=${SHOP_ENV_FILE:-/etc/shop-deploy.env}
ACTION=${1:-help}
case "$ACTION" in
  prepare|build|init|update|backup|status|logs) ;;
  *) echo '用法: sudo ./scripts/deploy.sh {prepare|build|init|update|backup|status|logs}'; exit 0 ;;
esac
[[ $EUID -eq 0 ]] || { echo '请使用 sudo 执行。' >&2; exit 1; }
command -v flock >/dev/null
exec 9>/var/lock/shop-deploy.lock
flock -n 9 || { echo '已有部署或备份正在执行。' >&2; exit 1; }
if [[ $ACTION == prepare ]]; then
  command -v python3 >/dev/null
  python3 - "$ROOT" "$ENV_FILE" <<'PY'
from pathlib import Path
import secrets, base64, os, sys
root, env = Path(sys.argv[1]), Path(sys.argv[2])
data = Path('/var/lib/shop')
if env.exists() or (data / 'config/config.yaml').exists():
    sys.exit('配置已存在，拒绝覆盖。请编辑现有文件。')
password = secrets.token_hex(32)
key = base64.b64encode(secrets.token_bytes(32)).decode()
for directory in ['config', 'geoip', 'uploads', 'backups', 'releases']:
    p = data / directory
    p.mkdir(parents=True, exist_ok=True)
    if directory in ['config', 'geoip', 'uploads']:
        os.chown(p, 10001, 10001)
        p.chmod(0o750)
config = (root / 'backend/configs/config.local.example.yaml').read_text()
config = config.replace('127.0.0.1:8080', '0.0.0.0:8080')
config = config.replace('host=127.0.0.1', 'host=postgres').replace('REPLACE_WITH_DATABASE_PASSWORD', password)
config = config.replace('upload_dir: ./storage/uploads', 'upload_dir: /var/lib/shop/uploads')
config = config.replace('trusted_proxies: [127.0.0.1/32, "::1/128"]', 'trusted_proxies: [172.30.80.0/24]')
config = config.replace('active_key_id: ""', 'active_key_id: primary').replace('keys: {}', 'keys: {primary: "' + key + '"}')
path = data / 'config/config.yaml'
path.write_text(config)
os.chown(path, 10001, 10001)
path.chmod(0o600)
env.parent.mkdir(parents=True, exist_ok=True)
env.write_text((root / 'deploy/env.prod.example').read_text().replace('REPLACE_WITH_RANDOM_HEX', password))
env.chmod(0o600)
print(f'已生成 {env} 和 {path}。请填写 WEBSITE_URL；按需配置 COS 和国家库。')
PY
  exit 0
fi
[[ -f $ENV_FILE ]] || { echo "先运行 prepare，缺少配置: $ENV_FILE" >&2; exit 1; }
# 不 source 配置，避免将变量值当作 shell 代码执行。
while IFS= read -r line || [[ -n $line ]]; do
  [[ -z $line || $line == \#* ]] && continue
  key=${line%%=*}; value=${line#*=}
  case "$key" in
    SHOP_DATA_DIR|POSTGRES_PASSWORD|WEBSITE_URL|HTTP_BIND|ADMIN_BIND) export "$key=$value" ;;
    *) echo "未知部署配置项: $key" >&2; exit 1 ;;
  esac
done < "$ENV_FILE"
: "${SHOP_DATA_DIR:?}" "${POSTGRES_PASSWORD:?}" "${WEBSITE_URL:?}"
[[ $SHOP_DATA_DIR == /* && $SHOP_DATA_DIR != / ]] || { echo '数据目录必须是非根目录的绝对路径'; exit 1; }
[[ $WEBSITE_URL =~ ^https?:// && $WEBSITE_URL != *YOUR_SERVER_IP* ]] || { echo '请先填写 WEBSITE_URL'; exit 1; }
command -v docker >/dev/null
docker compose version >/dev/null
docker info >/dev/null
export RELEASE_TAG=local
compose() {
  local -a files=(-f "$ROOT/deploy/compose.prod.yaml")
  # 服务器持久覆盖配置用于 HTTPS 等入口设置，所有维护命令使用同一配置。
  if [[ -f $SHOP_DATA_DIR/compose.override.yaml ]]; then
    files+=(-f "$SHOP_DATA_DIR/compose.override.yaml")
  fi
  docker compose --env-file "$ENV_FILE" "${files[@]}" "$@"
}
compose config --quiet
mkdir -p "$SHOP_DATA_DIR/releases" "$SHOP_DATA_DIR/backups"
CURRENT="$SHOP_DATA_DIR/releases/current"
if [[ -f $CURRENT ]]; then
  RELEASE_TAG=$(cat "$CURRENT")
  [[ $RELEASE_TAG =~ ^[a-zA-Z0-9_.-]+$ ]] || { echo '非法版本记录'; exit 1; }
fi
backup() {
  local destination
  destination="$SHOP_DATA_DIR/backups/$(date -u +%Y%m%dT%H%M%SZ)-$$"
  mkdir -m 700 "$destination"
  echo "备份到 $destination"
  compose exec -T postgres pg_dump -U admin -d peptide -Fc < /dev/null > "$destination/database.dump.partial"
  mv "$destination/database.dump.partial" "$destination/database.dump"
  tar -C "$SHOP_DATA_DIR" -czf "$destination/files.tar.gz" config geoip uploads releases
  cp "$ENV_FILE" "$destination/deploy.env"
  if [[ -f $SHOP_DATA_DIR/compose.override.yaml ]]; then
    cp "$SHOP_DATA_DIR/compose.override.yaml" "$destination/compose.override.yaml"
  fi
  if [[ -d $SHOP_DATA_DIR/nginx ]]; then
    tar -C "$SHOP_DATA_DIR" -czf "$destination/nginx.tar.gz" nginx
  fi
  if [[ -f $CURRENT ]]; then cp "$CURRENT" "$destination/release"; fi
  touch "$destination/COMPLETE"
}
build() {
  # 每次生成独立镜像标签，保留旧镜像供人工回退。
  RELEASE_TAG="$(date -u +%Y%m%dT%H%M%SZ)-$(git -c safe.directory="$ROOT" -C "$ROOT" rev-parse --short HEAD)-$$"
  export RELEASE_TAG
  for service in backend website nginx; do compose build "$service"; done
  printf '%s\n' "$RELEASE_TAG" > "$SHOP_DATA_DIR/releases/last-built"
  echo "构建完成: $RELEASE_TAG"
}
case "$ACTION" in
  build) build; exit 0 ;;
  backup) backup; exit 0 ;;
  status) compose ps; exit 0 ;;
  logs) compose logs --tail 100; exit 0 ;;
  update) [[ -f $CURRENT ]] || { echo '尚未成功部署，请使用 init'; exit 1; } ;;
  init) [[ ! -f $CURRENT ]] || { echo '已经部署，请使用 update'; exit 1; } ;;
esac
[[ -f $SHOP_DATA_DIR/config/config.yaml ]] || { echo '后端配置文件不存在'; exit 1; }
if [[ $ACTION == init ]]; then
  read -r -s -p '首位管理员初始密码: ' SEED_ADMIN_PASSWORD
  echo
  [[ ${#SEED_ADMIN_PASSWORD} -ge 12 && ${#SEED_ADMIN_PASSWORD} -le 72 ]] || { echo '密码须为 12–72 字节'; exit 1; }
  export SEED_ADMIN_PASSWORD
fi
# 构建失败不会停止当前服务。
build
compose up -d --wait --wait-timeout 180 postgres
stopped=0
on_error() {
  local code=$?
  if [[ $stopped == 1 ]]; then
    compose stop nginx website backend || true
    echo '发布失败，应用保持停止以避免旧代码访问部分迁移后的数据库。检查日志和备份后人工恢复；不要直接执行数据库 down migration。' >&2
  else
    echo '发布失败，请检查上方错误；尚未切换应用版本。' >&2
  fi
  exit "$code"
}
trap on_error ERR
# 先停入口和写入服务，备份完成后才开始迁移。
stopped=1
compose stop nginx website backend
backup
compose run --rm --no-deps backend -migrate
if [[ $ACTION == init ]]; then
  compose run --rm --no-deps -e SEED_ADMIN_PASSWORD backend -seed
  unset SEED_ADMIN_PASSWORD
else
  compose run --rm --no-deps backend -seed
fi
compose up -d --no-build --wait --wait-timeout 240 backend website nginx
# 所有容器健康后才更新成功版本记录。
printf '%s\n' "$RELEASE_TAG" > "$CURRENT.tmp"
mv "$CURRENT.tmp" "$CURRENT"
stopped=0
trap - ERR
compose ps
echo "部署成功: $RELEASE_TAG"
