# 管理后台与英文官网

单个 Go 服务提供公共 API、管理 API 和本地测试图片；PostgreSQL 存储用户、角色、会话、内容及媒体引用。后台使用 React / Umi Max / Ant Design，固定中文；官网使用 Next.js，固定英文。三端共用 `shared/contracts` 和 `shared/content`，需要一起安装、构建和发布。

已实现用户与角色权限树、密码加 TOTP 登录、首次改密绑定、敏感操作再认证、操作审计、COS 图片直传、目录管理和引用保护、自定义规格与 SKU、USD 精确金额、产品和文章富文本、未保存内容预览、官网搜索与分页、三个不限条数 Banner 位、品牌 Logo 与主题、完整 YAML 热加载。产品支持必填主图和选填检测报告图片，素材从媒体库选择；不含颜色、独立图库、库存或交易。

后台采用浅色导航和蓝色主色，产品支持名称、分类、状态筛选及分区编辑；新闻支持标题、状态、置顶筛选，并显示封面、排序与发布时间。筛选由服务端执行，支持刷新、分页和未保存修改提醒。

独立检测报告业务、询盘表单及分析脚本不在本次范围。联系入口使用后台客户配置，支持按国家 WhatsApp 分流；News 展示后台已发布文章，旧报告页及 Research 地址跳转到 News。官网品牌标记、页面标题与介绍正文可在品牌设置中维护，首页主视觉使用后台 Banner 投放。

问答配置位于后台「运营管理 → 问答配置」，支持新增、编辑、删除及按问题搜索。问题和答案为必填文本，支持排序和开启开关；官网独立 FAQ 页面（`/faq`）只展示开启且未删除的问答，按排序升序、相同排序按创建 ID 升序排列，答案保留换行。Home（`/`）依次展示 Banner、产品分类、精选产品、品牌介绍、新闻和联系咨询，不包含问答和品质检测区块。商品管理支持首页推荐开关、首页排序和推荐状态筛选；仅已发布且具有有效定价 SKU 的推荐商品进入首页，按首页排序、商品 ID 升序取前 6 个。分类、精选产品和新闻无数据时隐藏，品牌介绍使用首页标题及正文，新闻按现有置顶和排序规则取前 3 篇已发布文章。导航和页脚提供 FAQ 入口；无可展示问答时，FAQ 页面显示暂无内容提示。

数据库初始结构为 `backend/internal/data/migration/000001_schema.up.sql`，包含全部业务表及字段注释。空库执行 `go run ./cmd/server -migrate` 后，再执行 `-seed` 初始化权限、默认角色、币种和 Banner 位。项目按全新站点初始化，不提供旧版本结构的增量升级链。

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

管理后台二次验证由 YAML 的 `auth.totp.enabled` 控制，默认 `true`；设为 `false` 时只用密码登录，保留首次强制改密，敏感操作再认证仅校验密码。开关支持热加载，重新开启后关闭期间创建的密码登录会话需重新登录并验证。绑定接口仍可使用，关闭时仍须保留加密密钥配置。

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

本地运营管理测试数据：先在 `backend` 目录运行 `go run ./cmd/server -migrate` 应用已有迁移，再运行 `go run ./cmd/seed-demo` 查看数据数量，添加数据使用 `go run ./cmd/seed-demo -apply`。该命令只接受本机数据库连接，新增带 `[TEST]` 标记的媒体、分类、商品、新闻、首页 Banner、问答和关闭的客服配置；按固定名称或 slug 跳过已有测试记录，不覆盖原有内容。首页 Banner 在 `home_hero` 投放位启用两条测试内容，分别链接 Products 和 News，可在后台「Banner管理」中修改。客服号码和图片均为测试占位，不作为真实联系方式或二维码使用。商品和新闻包含发布、草稿及下架状态，问答包含开启和关闭状态，可用于验证官网可见性。

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

WhatsApp 国家配置、二维码、轮询、更新时间防覆盖及官网 H5 部署见 [WhatsApp 客服说明](docs/WHATSAPP_ROUTING.md)。本地官网 pnpm dev 通过 3000 入口覆盖访客 IP 请求头；美国默认配置种子不包含虚构联系方式，需在后台补齐真实账号。

后端配置模板中的服务监听、连接池、数据库超时、SQL 日志、素材和认证字段均附中文说明。数据库仅支持 PostgreSQL；日志输出到标准输出，等级由 log.level 控制。全量 SQL 需同时开启 data.database.log_sql 和 debug 等级，参数不写入日志。

超管初始化不读取运行 YAML：首次密码通过 SEED_ADMIN_PASSWORD 环境变量传入，用户名通过 -seed-admin-username 指定（默认 admin）。已有配置中的 seed 和 product 段应删除；商品币种直接使用 currency 表中的启用记录，默认取编码排序后的第一项，已有 SKU 不自动换币。

backend/bin/server 是 make build 生成的本机可执行文件，backend/bin/ 已由 Git 忽略。代码库只保存源码，部署构建产物通过发布流程交付。

媒体库仅支持图片，COS 配置、目录规则及上传接口见 [媒体库说明](docs/MEDIA_LIBRARY.md)。未配置 COS 时禁用新上传，已有图片与目录管理可继续使用。

首页推荐字段包含在初始结构中，商品默认不推荐，需在商品管理中开启「首页推荐」。商品与新闻均保留富文本正文和图片引用；官网页面标题及描述取商品名称／新闻标题和摘要。商品分类通过 category_id 关联，不存重复名称；商品不包含独立理化字段或自定义 SEO 字段。

官网桌面端右侧提供阅读尺标：刻度带随整页阅读进度连续滚动，章节标题随正文位置沿侧边移动，字母从词尾起依次沿向页面内侧弯曲的贝塞尔轨迹，在底部横排、侧边竖排和顶部横排之间过渡；下一章进入时，当前标题逐渐缩小并移入顶部队列，点击标题可跳转。章节来自实际展示的正文标题，排除商品卡片与弹窗；宽度小于 1200px 时隐藏，客服悬浮按钮避开侧栏，滚动动画遵循系统减少动态效果设置。
