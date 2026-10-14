#!/usr/bin/env sh
# Certbot deploy hook：证书续期成功后检查并重载容器 Nginx。
set -eu
/usr/bin/docker exec shop-nginx-1 nginx -t
/usr/bin/docker exec shop-nginx-1 nginx -s reload
