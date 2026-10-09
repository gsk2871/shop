# 外贸展示与询盘系统

三个独立目录：`backend/` 单个 Go 服务，`admin-fe/` 管理后台，`website/` 英文官网。每个站独立部署、独立数据库。使用 Kratos、Wire、GORM 与 PostgreSQL；后台沿用 admin-fe 的 React、Umi Max、Ant Design。官网使用 Next.js 服务端渲染。

## 已实现范围

- 系统管理：单管理员登录、退出、修改密码、最近操作日志。密码 bcrypt 哈希；8 小时数据库会话；修改密码使所有会话失效。
- 素材库：上传、列表、图片预览、复制引用地址。支持 JPG、PNG、WebP、GIF、PDF、MP4、WebM，单文件最大 20MB。无转码、VOD、字幕或回调业务。
- 文章管理：新增、编辑、删除；草稿、直接发布、下线；英文标题、Slug、摘要、正文、封面、SEO 字段。当前正文为纯文本，不是原项目富文本编辑器的完整迁移。
- 商品管理：新增、编辑、删除、草稿、上下架；名称、Slug、分类、别名、CAS、分子式、分子量、序列、纯度、规格、储存与溶解说明、摘要、详情和 SEO；最多 20 张共享商品图片，第一张为主图，无颜色与图片绑定。支持名称、别名、CAS、序列、分类搜索，列表最多返回 200 条。
- 官网配置：品牌名称、Logo、主色、背景色、公司介绍、邮箱、默认 WhatsApp、科研声明、最多 3 张 Banner。图片从素材库选取。
- 官网：首页、多 Banner 切换、文章列表及详情、公司介绍、联系页、科研声明、Cookie 偏好提示、响应式布局。只显示已发布文章。

官网产品展示页与搜索界面、报告业务列表、询盘表单和按国家 WhatsApp 分流尚未实现。商品后台和公共商品 API 已实现；其他未实现的后台入口仍为占位页，官网产品和报告页仍显示待完善状态。当前联系按钮使用后台配置的默认号码。Cookie 偏好保存在浏览器，尚无分析脚本及 Google Consent Mode 集成。隐私页为开发说明，正式发布前需要补充经营主体与数据处理信息。

## 本地启动

要求 Go 1.25+、Node.js 20.9+、pnpm 9.15、PostgreSQL（示例使用 Docker Compose 启动 PG 17）。

从项目根目录运行：

```sh
cp .env.example .env
cp backend/configs/config.local.example.yaml backend/configs/config.local.yaml
cp website/.env.example website/.env.local
docker compose up -d postgres
```

修改根目录 `.env` 的 PG 密码时，同步修改 `backend/configs/config.local.yaml` 的 `data.database.dsn`，将模板中的 `REPLACE_WITH_DATABASE_PASSWORD` 替换为实际密码。已有数据卷不会自动改变 PG 用户密码。

后端采用分层结构，只保留一个服务入口 `backend/cmd/server`。官网 API、管理 API、健康检查和素材访问共用一个 HTTP 服务。

```text
backend/
├── cmd/server/              # 单服务入口、Wire 装配
├── configs/
│   ├── config.local.example.yaml  # Git 提交模板，密码为占位符
│   └── config.local.yaml          # 本地实际配置，Git 忽略
└── internal/
    ├── conf/                # app/server/data/auth 配置类型、加载与校验
    ├── pkg/bootstrap/       # 配置加载、迁移、管理员初始化与服务启动
    ├── server/              # HTTP 服务、超时和路由装配
    ├── service/             # HTTP 接口
    ├── biz/                 # 业务模型、校验和仓储接口
    └── data/                # PostgreSQL、仓储及增量迁移
```

从后端目录启动时默认加载 `configs/config.local.yaml`，不会自动读取模板或 `.env`。配置按 `app / server / data / auth` 分组，包含服务名称、HTTP 超时、数据库连接池、上传目录及会话 Cookie。环境变量 `HTTP_ADDRESS`、`DATABASE_DSN`、`UPLOAD_DIR`、`COOKIE_SECURE` 优先于 YAML；未提供本地文件时可仅使用环境变量部署。生产指定独立配置：`go run ./cmd/server -conf /absolute/path/config.prod.yaml`，初始化和迁移同样支持 `-conf`。运行时 YAML 全部被 Git 忽略，仅 `*.example.yaml` 模板可提交。

也可在后端目录运行 `make init-config` 创建本地配置（不会覆盖已有文件），编辑后执行 `make migrate`、`make init-admin`、`make run`。通过 `make run CONF=/absolute/path/config.prod.yaml` 切换配置。

初始化数据库：

```sh
cd backend
go run ./cmd/server -migrate
```

创建管理员。密码不得少于 12 字节或超过 72 字节；通过交互读取，不写入仓库或命令行历史：

```sh
export ADMIN_USERNAME=admin
read -s ADMIN_PASSWORD
export ADMIN_PASSWORD
go run ./cmd/server -init-admin
unset ADMIN_PASSWORD
go run ./cmd/server
```

后端命令共用本地 YAML 配置；管理员账号与密码仍通过环境变量传入。`-init-admin` 不覆盖已有账户。

分别在其他终端运行：

```sh
cd admin-fe
pnpm install --frozen-lockfile
pnpm dev
```

```sh
cd website
pnpm install --frozen-lockfile
pnpm dev
```

- 管理后台：`http://localhost:8000`
- 官网：`http://127.0.0.1:3000`
- 后端：`http://127.0.0.1:8080`

先登录后台上传 Logo/Banner，再保存官网配置并发布文章，刷新官网即可看到新内容。未连接后端时，官网显示默认品牌文案和内容不可用提示，不伪造产品和文章。

## 构建与测试

```sh
cd backend
make wire
go test ./...
go vet ./...
make build
```

```sh
cd admin-fe
pnpm run setup
pnpm typecheck
pnpm build
```

```sh
cd website
pnpm build
pnpm typecheck
pnpm start
```

Next.js 构建后可用 `pnpm start` 启动正式官网服务。管理后台构建产物在 `admin-fe/dist/`，由 Nginx 提供静态文件；官网运行独立 Node.js 服务。

## 数据与部署

- `backend/internal/data/migration/000001_cms.up.sql` 为初始结构；对应 down 文件用于撤销，撤销会删除业务数据，勿对有效数据直接执行。
- `-migrate` 按文件名顺序执行所有内嵌 `.up.sql`，使用 `schema_migration` 记录已执行文件，事务与数据库锁防止重复执行。已有初始数据库可直接再次运行 `-migrate` 补建商品表。新增结构使用增量文件；没有自动执行 down 的命令。
- 不执行 AutoMigrate。Wire 已生成。业务 proto、OpenAPI、GORM Gen 尚未接入，不能把 `backend/api/` 当成已有接口契约。
- `UPLOAD_DIR` 默认为后端工作目录下的 `storage/uploads`。素材文件当前保存在本地磁盘，数据库仅保存元数据；迁移或备份必须同时处理两者。阿里 OSS / 腾讯 COS / AWS S3 尚未接入。
- 官网通过服务端 `BACKEND_URL` 获取配置与文章，并将 `/uploads/` 转发到后端。该变量不可指向管理后台静态服务。
- 后台开发代理 `/backend/`；生产代理示例在 `admin-fe/nginx.conf`，部署到不同主机时修改上游地址，并设置 `client_max_body_size`。
- `COOKIE_SECURE` 默认为开启，示例本地 HTTP 配置关闭。生产 HTTPS 必须设置 `COOKIE_SECURE=true`；管理 API 不开放跨域，写请求需要专用请求头和有效会话。
- 登录限流每 IP 每分钟 5 次，记录容量有上限。应用位于代理后面时当前使用代理连接 IP，需在明确可信代理列表后再实现真实访客 IP。
- 素材上传后可公开访问，只上传用于官网展示的公开资料。

## 故障排查

- 后端缺少 DSN：从 `backend/` 目录启动以加载 `configs/config.local.yaml`，或用 `-conf` 指定配置，也可导出 `DATABASE_DSN`。程序不自动读取 `.env`。
- 登录后仍返回 401：本地 HTTP 设置 `COOKIE_SECURE=false`，生产通过 HTTPS 访问。
- 找不到数据表：先执行 `-migrate`。
- 登录账号不存在：执行 `-init-admin`；无内置默认管理员。
- 上传返回 413：检查 Nginx 或其他入口代理的上传大小限制。
- 官网默认内容：检查后端是否启动、PG 连接是否可用、`BACKEND_URL` 是否正确。
- 文章不显示：确认状态为 `published`，Slug 合法且唯一。
