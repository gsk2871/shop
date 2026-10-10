# 管理后台与英文官网

单个 Go 服务提供公共 API、管理 API 和本地媒体；PostgreSQL 存储用户、角色、会话、内容及媒体引用。后台使用 React / Umi Max / Ant Design，固定中文；官网使用 Next.js，固定英文。三端共用 `shared/contracts` 和 `shared/content`，需要一起安装、构建和发布。

已实现用户与角色权限树、密码加 TOTP 登录、首次改密绑定、敏感操作再认证、操作审计、媒体流式上传和引用保护、自定义规格与 SKU、USD 精确金额、产品和文章富文本、未保存内容预览、官网搜索与分页、三个不限条数 Banner 位、品牌 Logo 与主题、完整 YAML 热加载。产品不含颜色、独立图库、库存或交易。

检测报告业务、询盘表单、按国家 WhatsApp 分流及分析脚本不在本次范围。联系页使用品牌配置中的默认联系方式，报告页提供文档询问入口。

## 本地启动

需要 Go 1.25+、Node.js 20.9+、pnpm 9.15 和 PostgreSQL 17。

```sh
cp .env.example .env
cp backend/configs/config.local.example.yaml backend/configs/config.local.yaml
cp website/.env.example website/.env.local
cp admin-fe/.env.example admin-fe/.env.local
docker compose up -d postgres
```

修改 `.env` 的数据库密码，同时填写后端私有 YAML 的 DSN。后端不会自动读取 `.env`，配置模板也不会自动用作运行配置。运行 YAML 不进入 Git。

为 `auth.totp.keys` 生成一个 32 字节密钥，并设置相应 `active_key_id`；本地 HTTP 使用 `cookie_secure: false`。密钥存入权限为 0600 的私有配置，和数据库、媒体目录配套备份。

```sh
openssl rand -base64 32
cd backend
go run ./cmd/server -migrate
read -s SEED_ADMIN_PASSWORD
export SEED_ADMIN_PASSWORD
go run ./cmd/server -seed
unset SEED_ADMIN_PASSWORD
go run ./cmd/server
```

`-conf /absolute/path/config.yaml` 可用于以上全部命令；`make migrate`、`make seed`、`make run` 支持 `CONF`。种子只创建权限树、缺失默认角色及授权、首位超管、USD 和 Banner 位。重复执行不重置现有密码、MFA、角色授权或账号状态，不生成演示产品、文章或投放。

首次登录进入安全中心：修改初始密码 → 重新输入新密码 → 扫描本地生成的验证器二维码并确认 → 重新执行密码和验证码登录。登录票据只保存在页面内存；完整会话使用 HttpOnly、SameSite=Strict Cookie。超级管理员创建其他用户时，用户也需完成首次设置。

在另外两个终端分别启动：

```sh
cd admin-fe
pnpm install --frozen-lockfile
pnpm run dev
```

```sh
cd website
pnpm install --frozen-lockfile
pnpm run dev
```

后台默认 `http://127.0.0.1:8000`，官网 `http://127.0.0.1:3000`，后端 `http://127.0.0.1:8080`。后台 `/backend` 代理保持原 Host；官网服务端读取 `BACKEND_URL`，公共请求不携带管理 Cookie。

## 验证

```sh
cd backend
make wire
go test ./...
go vet ./...
go build ./cmd/server
```

真实 PostgreSQL 验证须显式指定 `TEST_DATABASE_DSN`，测试账号需要创建／删除测试数据库的权限。测试为每个场景建立唯一临时数据库并清理，禁止指定生产实例。

```sh
TEST_DATABASE_DSN='host=127.0.0.1 port=5432 user=shop_test dbname=postgres sslmode=disable' go test -race ./...
```

```sh
cd admin-fe
pnpm run setup
pnpm run typecheck
pnpm run build
```

```sh
cd website
pnpm run typecheck
pnpm run build
```

根目录验证权限种子：`python3 scripts/check_permission_seed.py`、`python3 -m unittest discover -s scripts -p test_check_permission_seed.py`。

## 部署与配置

后端逐版准备独立数据库池、目录及监听；全部校验成功才发布。新请求使用新版本，在途请求和事务固定旧版本，旧资源排空后释放。修改失败保留最后可用版本，运行配置页可查看版本、成功时间、失败原因及字段来源；页面不返回密钥或数据库凭据。DSN 只允许切换到相同站点身份及相同 schema 的数据库。

环境覆盖在启动时固定捕获：`HTTP_ADDRESS`、`DATABASE_DSN`、`UPLOAD_DIR`、`COOKIE_SECURE`、`SEED_ADMIN_PASSWORD`。被覆盖的 YAML 字段不会取代环境值。需要运行时轮换的密钥放在受保护 YAML 文件中。

部署顺序、Nginx、密钥轮换、MFA 恢复及配套备份见 [CMS 接口与部署说明](docs/CMS接口与部署说明.md)。完整字段及实效证据见 [配置矩阵](docs/CONFIG_RELOAD_MATRIX.md)，接口规则见 [实施契约](docs/IMPLEMENTATION_CONTRACT.md)，本次验收见 [验收记录](docs/IMPLEMENTATION_ACCEPTANCE.md)。
