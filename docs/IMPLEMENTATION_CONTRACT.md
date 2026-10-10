# 管理后台与英文官网实施契约

这是全新项目改造的共同契约。后端、管理后台与官网按各计划步骤接入；本文件中的目标接口不能视为当前已可调用。没有旧业务数据，直接建设目标结构，不做数据搬迁和旧协议过渡。

## 三端共同类型

Go 定义位于 backend/internal/service/contract，TypeScript 的唯一公共类型来源位于 shared/contracts，由 admin-fe/src/services/contracts.ts 和 website/lib/contracts.ts 分别导出。管理权限类型仅管理后台导出。

| 对象 | 字段和规则 |
| --- | --- |
| Error | code、message、requestId；稳定 code 决定中英文提示，错误不返回数据库或凭据细节 |
| Page | items、total、page、pageSize；默认 20、最大 100，空 items 是数组；公共和管理列表都分页 |
| ID | 数据库 BIGINT 标识在新 DTO 中使用十进制字符串，避免 JavaScript 精度损失；不以 number 处理金额 |
| VersionedWrite | expectedVersion 为必填版本；版本初始 1，修改、状态操作及改价均递增；冲突返回 409 |
| Price | currency、salePrice、displayPrice；USD 12.34 对应入库 1234，换算依据 currency.minor_unit |
| RichText | html 与 mediaIds；只允许受控媒体节点引用媒体库 ID；服务端根据正文实际节点校验并提取引用，不信任声明的 mediaIds |
| PermissionNode | code、name、type、route、permissionCode、sort、children；只在后台传播 |

金额输入 salePrice 是十进制字符串，后端拒绝负值、超精度、指数表示与溢出。返回金额按币种精度保留小数；客户端显示 displayPrice。ID 字符串、分页与新错误响应在对应模块三端一起接入，不对现有处理器提前强行换协议。

媒体节点示例为 img 或 video 上的 data-media-id 属性。客户端不可提交任意外部 src 来代替选库；服务端根据 ID 生成可用地址。引用 ID 列表必须与正文节点完全一致，HTML 净化不能保留未知引用。预览执行同一净化与金额处理流程，但不保存内容或持久引用。

## 当前与目标接口

管理前缀为 /admin/api/v1；官网前缀为 /api/v1。后台开发代理 /backend，官网服务器只读取公共接口，Cookie 和后台权限树不传给官网。

| 领域 | 当前接口 | 目标接口与权限 | 后台调整 | 官网调整 |
| --- | --- | --- | --- | --- |
| 登录 | POST login；GET me；POST logout/password | 新增 POST login/totp、totp/setup、totp/activate、GET me/permissions；按身份和 scope 校验 | 两阶段登录、改密、绑定、菜单裁剪 | 公共客户端隔离后台会话，英文降级状态 |
| 用户角色 | 无 | users、roles、permissions、roles/{id}/permissions；完整目录树种子 | 用户、角色、树形授权页 | 公共 DTO 不携带管理字段 |
| 审计 | GET audit | 分页，audit.read | 筛选及中文操作名称 | 公共错误 requestId 可用于查询，审计不公开 |
| 配置状态 | 无 | GET config/status，config.read | 版本与加载状态 | 公共内容请求及媒体请求适配资源切换 |
| 媒体 | GET/POST media，GET/HEAD /uploads/{name} | 详情、重命名、删除、references、upload-policy，按媒体动作授权 | 分页、预览、选库、限制刷新 | 媒体播放和 Range 请求 |
| 产品 | GET/POST/PUT products、DELETE products/{id} | PUT products/{id}、状态 PATCH、SKU 改价、POST products/preview；发布和改价另验权限 | 规格、SKU、富文本、预览 | 分页搜索、规格选择、价格与详情 |
| 新闻 | GET/POST/PUT articles、DELETE articles/{id} | PUT articles/{id}、状态 PATCH、POST articles/preview；发布另验权限 | 共用编辑器及媒体选择 | /news 列表和详情富文本 |
| 主题 | GET/PUT settings | version、主题和 logo_media_id，site.read/update | 主题及 Logo 配置 | 所有业务色消费主题变量 |
| Banner | site_config 内数组 | banner-slots、banners、banners/{id}/enabled；不限条数 | 分位置配置、排序、排期 | 按位置获取全部在线内容 |
| 金额 | 无 | GET currencies、SKU 价格，product.price.update | 字符串输入 | 显示后端起价和选中 SKU 价格 |

管理路由逐条声明权限，未登记业务路由默认拒绝。产品整份保存涉及价格或发布变化时仍检查相应动作权限，新闻与 Banner 同理。GET 详情与静态 preview、upload-policy 等路径必须显式注册，不能宽泛前缀匹配。

## 权限种子来源

可提交的唯一种子来源为 backend/internal/data/seed/admin_permissions.json。使用 python3 scripts/check_permission_seed.py 校验，测试命令为 python3 -m unittest discover -s scripts -p test_check_permission_seed.py。数据库加载器校验并事务导入，保留已有账号及角色授权。

现有种子包含 48 节点、43 业务权限和三种角色，包含完整 API 映射和自助 scope 清单。校验拒绝孤立父节点、环、重复代码、重复 API、未知角色授权、查看者写权限和额外匿名写入口。目录及角色按 code 幂等初始化，不产生演示业务内容。

## 工程规则与交付边界

每步同时登记 backend、admin-fe、website 的实际工作与验证。数据库采用单数表名、identity 主键、软删除与部分唯一索引；参照 db-schema-design 技能使用逻辑关系，由业务校验和事务维护引用，首次结构不加物理外键。不引入多语种表、商城库存或颜色图库。

目前未接入 proto、OpenAPI、GORM Gen，继续手写类型和显式 SQL 版本脚本，避免为套用参考项目命令引入整套工具链。新增表和 DTO 都应同步此契约、正式模块文档及对应三端实现。SQL 仅应用到独立开发或测试实例，不直接操作生产。

## 逐路由管理接口登记

下面的路径均以 /admin/api/v1 为前缀，均已接入。列表 GET 使用 Page；详情 GET 返回单个对象；创建 POST 返回 201 和新对象，更新返回 200 和更新后的对象，删除返回 200 和 {ok:true}。修改已有聚合时提交 expectedVersion；创建和无状态预览不要求它。媒体重命名、用户安全操作与角色授权以各模块的并发规则为准，不把内容版本号用于登录票据消费。

| 方法 | 路径 | 必需权限 | 计划步骤 |
| --- | --- | --- | --- |
| GET | `/media` | media.read | 06 |
| GET | `/media/{id}` | media.read | 06 |
| GET | `/media/{id}/references` | media.read | 06 |
| GET | `/media/upload-policy` | media.read | 06 |
| GET | `/products` | product.read | 09 |
| GET | `/products/{id}` | product.read | 09 |
| GET | `/currencies` | product.read | 08 |
| GET | `/articles` | article.read | 11 |
| GET | `/articles/{id}` | article.read | 11 |
| GET | `/settings` | site.read | 12 |
| GET | `/banner-slots` | banner.read | 12 |
| GET | `/banners` | banner.read | 12 |
| GET | `/banners/{id}` | banner.read | 12 |
| GET | `/users` | user.read | 04 |
| GET | `/users/{id}` | user.read | 04 |
| GET | `/roles` | role.read | 04 |
| GET | `/roles/{id}` | role.read | 04 |
| GET | `/permissions` | role.read | 04 |
| GET | `/roles/{id}/permissions` | role.read | 04 |
| GET | `/audit` | audit.read | 04 |
| GET | `/config/status` | config.read | 13 |
| POST | `/media` | media.upload | 06 |
| PATCH | `/media/{id}` | media.update | 06 |
| DELETE | `/media/{id}` | media.delete | 06 |
| POST | `/products` | product.create | 09 |
| PUT | `/products/{id}` | product.update | 09 |
| POST | `/products/preview` | product.preview | 09 |
| PATCH | `/products/{id}/status` | product.publish | 09 |
| DELETE | `/products/{id}` | product.delete | 09 |
| PATCH | `/products/{id}/skus/{skuId}/price` | product.price.update | 09 |
| POST | `/articles` | article.create | 11 |
| PUT | `/articles/{id}` | article.update | 11 |
| POST | `/articles/preview` | article.preview | 11 |
| PATCH | `/articles/{id}/status` | article.publish | 11 |
| DELETE | `/articles/{id}` | article.delete | 11 |
| PUT | `/settings` | site.update | 12 |
| POST | `/banners` | banner.create | 12 |
| PUT | `/banners/{id}` | banner.update | 12 |
| PATCH | `/banners/{id}/enabled` | banner.publish | 12 |
| DELETE | `/banners/{id}` | banner.delete | 12 |
| POST | `/users` | user.create | 04 |
| PUT | `/users/{id}` | user.update | 04 |
| PATCH | `/users/{id}/status` | user.disable | 04 |
| DELETE | `/users/{id}` | user.delete | 04 |
| POST | `/users/{id}/reset-password` | user.reset_password | 04 |
| POST | `/users/{id}/reset-totp` | user.reset_totp | 04 |
| POST | `/roles` | role.create | 04 |
| PUT | `/roles/{id}` | role.update | 04 |
| DELETE | `/roles/{id}` | role.delete | 04 |
| PUT | `/roles/{id}/permissions` | role.grant | 04 |

角色启停使用 PUT /roles/{id} 的 enabled 字段并校验 role.update，不额外增加未在权限种子登记的路径。用户启停统一使用 user.disable。静态 /media/upload-policy、/products/preview 和 /articles/preview 使用精确 method/path 注册。产品和新闻旧 PUT 集合路径在模块切换时删除。

匿名管理入口仅 POST /login 和 POST /login/totp。自助入口 GET /me、GET /me/permissions、POST /password、POST /logout 支持 full、onboarding、password_change；POST /totp/setup 和 POST /totp/activate 支持 full、onboarding，且再次检查未绑定及首次改密已完成。绑定成功撤销受限会话，返回重新登录指令。正式业务 API 仅允许 full。

## 公共接口及预览边界

| 方法与完整路径 | 返回目标 | 官网页面及要求 |
| --- | --- | --- |
| GET /api/v1/site | 品牌与文字标记、解析后 Logo URL、主题、公司介绍、官网页面标题与正文、Cookie 提示及兼容联系方式 | layout、首页、about、privacy、products、news；不返回后台 version、操作者或种子信息 |
| GET /api/v1/products | Page<产品摘要>，含服务端 fromPrice | /products，q 搜索，published 且有有效定价 SKU |
| GET /api/v1/products/{slug} | 产品公开详情、specs、有效 skus、detail | /products/[slug]，下架 404；无图库和管理字段 |
| GET /api/v1/articles | Page<新闻摘要> | /news，只读 published，置顶、sort、首次发布时间、ID 稳定排序 |
| GET /api/v1/articles/{slug} | 新闻公开详情、净化 body、封面 URL、SEO | /news/[slug]，草稿或下线 404 |
| GET /api/v1/banners?slot={code} | {items:在线 Banner[]} | home_hero、products_header、research_header；此位置读取特意不分页，不截断 |
| GET/HEAD /uploads/{storageRoot}/{storageKey} | 文件字节及 Range 206 | 稳定 root ID 与文件 key 解析，不拼接当前目录；两端代理保留路径和 Range |

分页参数 page、pageSize 必须为正整数，默认 1、20，超过 100 或非法数字返回 400 INVALID_ARGUMENT；total 为空时为 0，items 为 []。排序参数 sort、order 按各模块白名单检查；不将原始值拼到 SQL。时间输出 RFC3339 UTC，Banner 输入须带时区；后台按 Asia/Shanghai 展示。

产品和新闻预览 POST 返回与公共详情一致的展示字段，不含管理状态、数据库操作者或持久链接；未保存的产品规格与 SKU 使用表单内临时 key 关联，不伪造数据库 ID。预览仍执行权限、媒体可用性、HTML 净化和金额校验，不保存、不发布、不产生持久引用。预览屏幕宽度由后台本地切换，不向官网公开预览接口。

## 错误码登记

| HTTP | code | 使用边界 |
| --- | --- | --- |
| 400 | INVALID_ARGUMENT | 格式、分页、枚举、金额、非法正文或 ID 归属 |
| 401 | AUTHENTICATION_REQUIRED / INVALID_CREDENTIALS / MFA_INVALID / MFA_TICKET_INVALID | 会话缺失或失效；登录失败统一提示，票据过期、消费或失效不泄漏账户状态 |
| 403 | PERMISSION_DENIED / ONBOARDING_REQUIRED / REAUTHENTICATION_REQUIRED | 越权、未完成安全引导、敏感操作缺近期验证 |
| 404 | NOT_FOUND | 不存在或不能公开读取的内容 |
| 409 | VERSION_CONFLICT / MEDIA_IN_USE / CONFLICT / LAST_SUPER_ADMIN | 并发内容写入、引用保护、唯一键冲突、超管保护 |
| 429 | RATE_LIMITED | 登录与验证码尝试限流 |
| 500 | INTERNAL_ERROR | 未预期错误，内部细节仅关联 requestId 记录 |
| 503 | SERVICE_UNAVAILABLE | 数据库或运行资源暂时不可用，不旁路认证 |

HTTP 生成或复用经过格式校验的请求 ID，在响应 X-Request-ID 与错误 requestId 中一致返回。后台逐 code 映射中文，官网映射英文；未识别 code 使用通用提示。目标类型尚未替换当前 {error} 响应，各模块切换时同步客户端。

## 富文本编辑及共享渲染决策

采用 Lexical React 编辑器，后台封装工具栏、链接以及媒体库按钮；图片与视频使用自定义媒体节点，导出受控 HTML。所有 Lexical 包使用相同精确版本。当前验证候选为 0.52.0，React 19.2.5/19.3.0、TypeScript 6.0.3；最终锁文件在步骤 07 接入时保存。

依据 [Lexical React 官方指南](https://lexical.dev/docs/getting-started/react) 和 [React 包说明](https://lexical.dev/docs/packages/lexical-react)，编辑器支持自定义 React UI 与节点。编辑器仅在后台客户端加载。官网和后台预览复用 shared/content 下的 React 渲染组件和样式，不引入编辑器运行时到官网正文展示；官网 SKU 交互是独立 Client Component。Umi 与 Next.js 分别编译共享源码，不在运行时相互调用。

正文协议为 {html,mediaIds}。允许 p、h2/h3/h4、strong、em、u、ul/ol/li、blockquote、br、a、img、video；不允许 script、style、iframe、事件属性和任意 CSS。链接只接受受验证的站内路径或 https 地址；外链设置安全 rel。媒体节点仅接受 data-media-id，服务端解析媒体库后生成 src；video 使用 controls、preload=metadata，不允许任意 poster 或外部 source。复制粘贴的外部 img/video 不成为有效媒体。媒体类型必须与标签匹配，重复 ID 归一化，声明列表与实际节点集合不一致返回 INVALID_ARGUMENT。最终净化白名单和节点导入导出在步骤 07 测试固定。

## 完整实现登记

后台预览与官网详情使用 @shop/content 0.1.2，类型来自 @shop/contracts 0.1.2。产品公共 SKU 仅包含 id/key/specValueKeys/price；公共规格和值不包含 sort，顺序由后端数组确定。新闻公共 DTO 不包含 status/version/coverMediaId；Banner 公共 DTO 不包含启用开关、排期或素材管理 ID；品牌公共接口不含配置版本及 Logo 管理 ID。

POST /me/reauth 为 full scope 自助接口，接收 password/code；独立于角色业务权限，为敏感用户重置建立五分钟身份验证。配置运行状态需要 config.read，所有业务写入口均经精确方法和路径授权。完整字段行为与验收证据见 CONFIG_RELOAD_MATRIX.md 和 IMPLEMENTATION_ACCEPTANCE.md。

## WhatsApp 地域客服

国家配置、二维码、更新时间防覆盖、默认唯一及官网 GET／H5 契约见 [WhatsApp 国家客服分配](WHATSAPP_ROUTING.md)。
