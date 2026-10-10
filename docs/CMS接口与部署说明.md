# CMS 接口与首次部署

管理接口前缀为 `/admin/api/v1`，公共接口前缀为 `/api/v1`。后台通过 `/backend` 代理访问；公共媒体路径为 `/uploads/{rootId}/{storageKey}`。所有实体 BIGINT ID 对前端使用十进制字符串；错误固定为 `{code,message,requestId}`，不返回 SQL、密码或连接串。

## 身份与权限

`auth.totp.enabled` 默认 true，可通过配置文件热加载开启或关闭。关闭时密码正确即签发会话，未完成首次改密者仍只能访问安全中心；已绑定用户也跳过验证码，绑定／激活接口继续可用。敏感操作再认证在关闭时只验证当前密码，开启时要求密码与验证码。公共身份 `/me` 返回 `totpEnabled` 供后台界面使用。重新开启后，关闭期间的密码登录会话返回 401，需重新登录，未绑定者进入绑定流程。加密密钥仍必填并检查历史密文，关闭开关不清除绑定。

开关开启时，`POST /login` 接收 username/password。未完成设置者只得到受限 Cookie；已绑定者得到短期 ticket，不签发完整 Cookie。`POST /login/totp` 接收 ticket/code，一次性消费票据及验证码时间步后才签发完整会话。`GET /me` 返回用户、角色、scope、授权及供后台直接消费的菜单树。

自助接口为 `POST /logout`、`POST /password`、`POST /totp/setup`、`POST /totp/activate` 和 `POST /me/reauth`。初始化二维码在本地生成。受限 scope 不能调用业务接口；改密、停用、角色修改、重置 MFA 均撤销相关会话和票据。重置他人密码／MFA 要求 5 分钟内的密码加新验证码再认证和操作原因。并发操作不能移除最后一位有效超管。

完整路由与权限映射以 `backend/internal/data/seed/admin_permissions.json` 为唯一来源。包含 52 个节点、47 个业务权限、super_admin / content_admin / viewer 三种默认角色。未登记的业务路径默认拒绝；整份内容 PUT 也不能绕过独立的发布或价格权限。所有管理写请求要求 `X-Requested-With: peptide-admin`，并校验同源 Origin；仅信任配置中的代理 CIDR。

## 内容契约

列表使用 `{items,total,page,pageSize}`，页码从 1 开始，单页 1–100；产品与媒体支持 q 搜索。详情返回单对象；创建 POST 返回 201；更新返回 200；删除返回 `{ok:true}`。分类、产品、文章、Banner 和品牌设置修改要求 expectedVersion；冲突返回 409 VERSION_CONFLICT，页面需刷新重新编辑。

产品由基本信息、specs、values、skus、价格及 detail 组成。规格维度不超过 6 个，每维不超过 20 值，SKU 不超过 500 个；同产品组合不能重复，每个 SKU 必须从每维选择一个值，无规格产品使用一个默认 SKU。更新保留已有实体 ID，并校验规格／值／SKU 属于当前产品。产品状态单独 PATCH `/products/{id}/status`，SKU 价格 PATCH `/products/{id}/skus/{skuId}/price`。价格输入为 salePrice 十进制字符串，后端按 currency.minor_unit 转为整数，并返回格式化 price 和 fromPrice；首次种子补齐 USD，不换汇，不使用浮点。币种选项直接读取 currency 表中全部启用记录，默认取编码排序后的第一项；增加币种无需修改运行配置，已有 SKU 币种不自动改变。

商品分类管理使用 GET/POST `/categories`、GET/PUT/DELETE `/categories/{id}`，分别受 category.read/create/update/delete 权限控制。分类名称去首尾空白、不区分大小写唯一，sort 越小越靠前，同序按 ID 排列；删除为软删除，被任意未删除商品使用时返回 409 CATEGORY_IN_USE。GET `/categories/options` 由 product.read 授权，返回全部可选分类的 `{items:[{id,name}]}`；创建及编辑保存商品必传 categoryId，分类名称由服务端解析，重命名即时反映在商品展示中。

公共 GET `/api/v1/categories` 返回全部未删除分类的 `{items:[{id,name}]}`，不包含管理版本信息。GET `/api/v1/products` 的 categoryId 可选，不传或空值不增加分类筛选；q 仅对产品名称做不区分大小写的字面子串查询，两个条件可组合，total 与分页使用同一筛选。官网首次进入选中 All products，搜索／切换分类重置页码，翻页保留条件；分类与结果数量来自后台接口。

升级须先执行 000013_product_category migration，再运行 seed 更新菜单／权限登记，最后启动新版本。已有非空分类文字按去空白、忽略大小写归并为分类并回填 category_id；无分类的历史商品保持未分配，可出现在全部查询中，编辑保存须补选分类。种子不生成演示分类；超管自动补齐新权限，已有普通角色需在角色管理中授权。开发回退 down SQL 会将分类当前名称写回商品原 category 字段。

产品 detail 和文章 body 为 `{html,mediaIds}`。HTML 只保留段落、h2–h4、强调、列表、引用、安全链接和受控图片／视频。媒体节点含 `data-media-id`，必须对应可用素材及后端生成的 URL；正文 ID 集合必须一致。脚本、iframe、SVG、事件属性和外部媒体会被删除或拒绝。`POST /products/preview`、`POST /articles/preview` 对当前表单进行同一套净化，返回公共 DTO，不保存、发布或创建持久引用。后台预览和官网共用 `@shop/content`。

文章保存、发布、删除与正文／封面引用在同一事务完成。首次发布后保持 firstPublishedAt；公共列表按置顶、sort、首次发布时间及 ID 稳定排序，草稿、下线、软删除返回 404。公共产品只返回已发布且有启用定价组合的内容；公共 DTO 排除版本、状态、编辑者和 SKU 管理字段。

Banner 位仅种子 home_hero、products_header、research_header。投放字段为位置、英文标题／副标题、桌面图片 ID、可选手机图片 ID、英文 CTA、站内或 HTTPS 链接、sort、enabled、startsAt、endsAt 和 version。管理列表分页，公共 `GET /banners?slot=...` 返回全部在线投放，不设置条数上限；开始含边界、结束不含边界，数据库按 UTC 判断，后台按 Asia/Shanghai 编辑。手机图片缺失使用桌面图片。

品牌配置含 brand、logoMediaId、theme、description、email、whatsapp、disclaimer 和 version。theme 的 primary/background/surface/text/muted/border/accent/onPrimary 均为 #RRGGBB，官网主要区域使用统一 CSS 变量。后台支持颜色预览及恢复默认值。email 与 whatsapp 为历史兼容字段，官网不读取它们作为联系入口，品牌编辑界面不再显示这两项。无投放时省略投放区，首页不使用写死的默认主视觉。

## 媒体

`POST /media` 使用 multipart 的 file 字段，流式写入临时文件；内容嗅探、图片解码或视频容器校验成功后原子改名，最后事务写元数据和审计。默认图片／PDF 20 MiB、视频 200 MiB，可通过私有 YAML 和 `/media/upload-policy` 更新。支持 JPEG、PNG、WebP、GIF、PDF、MP4、WebM，不转码。`GET /media/{id}/references` 显示引用；PATCH 重命名；DELETE 检查引用并加锁，使用中返回 MEDIA_IN_USE。逻辑删除立即禁止公共访问，物理删除失败由任务重试。24 小时后的崩溃临时文件或无元数据文件会清理，在途上传路径受保护。

公共 GET/HEAD 支持视频 Range/206，校验素材可用性，设置 nosniff 与 sandbox CSP。上传目录改变后，历史 root 注册仍保留；文件不会自动搬迁。上传内容公开可访问。数据库、全部 storage_root.path 目录和历史加密密钥必须配套备份。

## 首次部署

1. 配套构建 backend、admin-fe、website，使用锁文件安装，保留 shared 两个包。使用 `backend/configs/config.local.example.yaml` 配置私有运行文件，填写 DSN、持久目录、TOTP active key 和 keyring，权限 0600。
2. 执行 `server -conf /path/config.yaml -migrate`，然后通过 SEED_ADMIN_PASSWORD 注入初始密码，执行 `-seed -seed-admin-username admin`；用户名也可指定其他合法名称，运行 YAML 不保存用户名和密码。再执行 `server -conf /path/config.yaml`；运行不会自动 seed 或 reset。
3. 后端仅内网监听，官网 Next 服务端使用 BACKEND_URL；后台构建前设置 UMI_APP_WEBSITE_URL 为官网地址；生产 cookie_secure=true，并填写实际反向代理 CIDR。后台浏览器与其 `/backend` 代理保持同源，Nginx 传递 Host、X-Forwarded-For 和 X-Forwarded-Proto。
4. 超管完成首次改密及 MFA，再创建运营和查看者账号。检查 readyz、运行配置版本和上传策略；创建业务内容之前官网显示默认品牌及英文空状态。
5. 官网 output=standalone，生产运行 `.next/standalone/server.js`，并将 `.next/static`、public（若存在）复制到对应位置；后台 dist 静态发布，未知前端路径回退 index.html。

后端上传默认视频上限为 200 MiB，Nginx 上传入口至少允许 201 MiB multipart 请求，并匹配实际配置的读取／代理超时。例如以下后台代理片段；官网 `/uploads/` 同样代理后端以保持 Range。

```nginx
location /backend/ {
    client_max_body_size 201m;
    proxy_read_timeout 100s;
    proxy_send_timeout 100s;
    proxy_set_header Host $http_host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_pass http://127.0.0.1:8080/;
}
location /uploads/ {
    proxy_pass http://127.0.0.1:8080;
}
```

Nginx、前端环境变量和官网 BACKEND_URL 使用各自发布机制，后端 watcher 不修改这些外部服务。实际生产发布不包含在本次改造中。

## 密钥与恢复

轮换先增加新密钥，保留所有旧密钥，再切换 active_key_id。新绑定使用新密钥；已有绑定继续读取原密钥。移除密钥前必须确认没有对应用户／待绑定密文，也没有使用该密钥的在途旧版本；否则整版热加载拒绝。当前不自动重加密全部账号，可通过受控重绑逐步迁移。

无法使用验证器时，在受控服务终端运行：

```sh
server -conf /path/config.yaml -recover-totp-user admin -recover-reason '经过身份核对的恢复原因'
```

命令撤销该用户会话和票据、清除绑定，并记录系统审计及原因；用户重新登录并绑定。无需修改源码、写入明文密钥或关闭 MFA。

配套恢复流程：停止后端写入 → pg_dump 自定义格式备份 → 备份每个媒体根目录及私有配置／全部历史密钥 → 在独立数据库 pg_restore → 将文件恢复到原注册路径（或受控调整注册目录后核对所有历史 root）→ 保留 site_identity → 验证 seed 幂等和启动 schema 检查 → 验证登录、旧文件、Range、已发布内容和加载状态。不要在文件未恢复时启动清理任务。down SQL 仅用于受控开发回退，不能代替配套备份恢复。

## WhatsApp 客服入口

客服配置、美国默认种子、国家数据库、IP 请求头、官网入口代理及 H5 验收见 [WhatsApp 国家客服分配](WHATSAPP_ROUTING.md)。本地官网使用 pnpm dev 的 3000 可信入口；生产必须在官网入口覆盖 X-Client-IP 并限制后端 trusted_proxies。

## 官网导航与内容配置

官网导航为 Home、Products、News、About。旧 /reports 和 /research 地址重定向到 /news；旧 /research/{slug} 重定向到 /news/{slug}。News 展示已发布文章，正文及封面在「文章管理」维护；列表按现有置顶、排序和发布时间规则展示，页头沿用 research_header 投放位。

- Home 的主视觉使用「Banner 投放」中的 home_hero；Products 页头使用 products_header。CTA 仅在投放配置了 ctaText 时展示，图片、标题、副标题和跳转地址均来自投放配置。
- 「品牌、主题与官网内容」维护 brand（站点名称及 SEO 标题）、brandMark（无 Logo 时的文字标记）、logoMediaId、description（SEO 简介及页脚）、theme。
- 同一设置页面维护 homeTitle/homeContent、aboutTitle/aboutContent、productsTitle、newsTitle、privacyTitle/privacyContent、cookieTitle/cookieContent。正文为纯文本，保留换行，HTML 按文本显示；正文和品牌标记留空时不展示，不补写示例宣传内容。Cookie 标题和正文均留空时不弹出提示。
- 设置仍由 GET/PUT /admin/api/v1/settings 管理，公共 GET /api/v1/site 返回上述展示字段。沿用 site_config.payload JSON 存储及版本冲突校验，无需数据库结构迁移。旧配置缺失新增字段时使用空值；升级后请在后台填写所需正文。
- 官网不再展示 disclaimer、固定科研声明及 DB-IP 链接；disclaimer 保留为兼容字段。导航、搜索、分页、联系按钮和错误提示属于界面操作文字。

## 运行配置

完整中文字段说明见 backend/configs/config.local.example.yaml，字段校验与消费者见 CONFIG_RELOAD_MATRIX.md。旧私有配置删除 seed 和 product 段后再启动本版本；加载器会拒绝未知字段。数据库连接串、连接池、连接空闲回收、建连／SQL 执行／等锁超时、慢查询阈值和 SQL 日志开关均由 data.database 配置；连接超时和 SQL 超时优先于 DSN 中对应设置。日志统一输出到标准输出，采用文本格式，等级支持 debug/info/warn/error 热加载。
