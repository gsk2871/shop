# 改造步骤 01 基线与交付记录

日期：2026-10-09。基线提交：cf8324f。采用本地《管理后台与英文官网分步骤改造计划》的编号及依赖；三端逐模块直接切换新契约，不增加历史数据兼容层。

## 本轮范围和工作区

开始检查时已有未提交修改：.gitignore、README.md、backend/.env.example、backend/configs/config.local.example.yaml、compose.yaml；未跟踪资产包括两端 contracts 导出、backend service/contract、权限种子、scripts、shared 和两份契约/配置文档。本轮保留这些内容，在现有契约和矩阵上补充实施登记，不将已有配置模板修改归为本轮新实现。

当前运行实现仍为单管理员、非用户归属会话、旧 error 响应、数组列表、文本产品及新闻、产品图库、最多三张 Banner。公共产品 DTO 直接使用 biz.Product，文章 DTO 直接使用 biz.Article，含 status 等后台字段。新版共享类型仅被两端导出，尚未由运行接口消费。步骤 01 的契约与种子登记不表示 RBAC、MFA 或热加载已完成。

## 基线验证

| 检查 | 实际结果 |
| --- | --- |
| backend go test ./... | 通过；使用 GOCACHE=/private/tmp/shop-go-build |
| backend go vet ./... | 通过；同上 |
| backend make build | 通过，产物 backend/bin/server |
| admin-fe pnpm run setup | 通过，生成 Umi 本地类型 |
| admin-fe pnpm typecheck | 通过 |
| admin-fe pnpm build | 通过，Umi 4.7.23；有 Node localStorage 实验提示，无构建错误 |
| website pnpm build | 通过，Next.js 16.4.0；当前没有 /products/[slug] 页面 |
| website pnpm typecheck | 通过 |
| python3 scripts/check_permission_seed.py | 通过，43 节点、38 权限、3 角色 |
| python3 -m unittest discover -s scripts -p test_check_permission_seed.py | 通过，2 个测试方法，负向子场景包含环、孤立节点、越权授权和非法匿名写入口 |
| 权限 JSON 来源一致性 | backend/internal/data/seed/admin_permissions.json 与本地计划引用 JSON 解析后完全相同 |

首次 Go 检查因默认缓存目录受沙箱限制失败，换临时缓存后全量通过，属于执行环境限制。未执行 migrate、seed、真实 PostgreSQL 集成测试、浏览器联调、上传文件及部署。make wire 本轮未执行：未改变构造函数，最终交付按计划补全。

## 当前路由、数据与代理

管理前缀 /admin/api/v1：POST login；GET me；POST logout/password；GET audit；GET/PUT settings；GET/POST/PUT products 和 articles；DELETE products/{id}、articles/{id}；GET/POST media。现有 PUT 集合路径在相应步骤切换为 PUT /{id}，发布和价格从整份编辑中独立授权。

公共前缀 /api/v1：GET site、products、products/{slug}、articles、articles/{slug}。基础设施 GET /healthz、/readyz 保留。当前 GET/HEAD /uploads/{name} 由 ServeFile 提供文件；官网 Next rewrite 转发 /uploads，后台 assetURL 通过 /backend/uploads 访问，后台 API 使用 /backend/admin/api/v1。新版存储路径继续经过同一代理，验证 Range 请求及 206。

当前数据库以完整初始结构 000001_schema 为准，up/down 成对，执行记录由 schema_migration 保存，事务 advisory lock 串行化初始化。不执行 AutoMigrate，无 proto/OpenAPI/GORM Gen 前置要求。

## 三端任务矩阵

| 步骤 | backend | admin-fe 中文后台 | website 英文官网 |
| --- | --- | --- | --- |
| 01 | DTO、路由权限、schema/种子登记与配置矩阵 | shared/contracts 导出、服务和菜单任务登记 | 公共类型导出、公开路径/代理/空状态登记 |
| 02 | configreload 快照、资源 Prepare/Commit/Rollback/Drain；站点/root 标识 | 配置状态服务与错误显示；上传策略在 06 消费 | 公共请求失败恢复、稳定 root 媒体路径 |
| 03 | admin_user/role/permission/role_permission、会话、审计及幂等 seed | 用户/菜单类型和初始化身份返回 | 空库公共读取与管理信息隔离 |
| 04 | 默认拒绝路由注册、每请求授权、超管并发保护 | 用户/角色/审计页面、树菜单与按钮 | 新公共错误契约，公共请求不携带管理凭据 |
| 05 | MFA ticket/secret/timestep、keyring、限流及恢复 | 密码→验证码、首次改密/绑定、安全中心 | 管理认证与官网请求隔离及英文错误 |
| 06 | 流式上传、upload-policy、media/reference、锁与清理 | 分页、进度、预览、重命名和 MediaSelect | 图片视频、Range、资源不可用提示 |
| 07 | HTML 净化、媒体引用提取、只读预览 DTO | Lexical 编辑器、受控选库节点、预览框架 | shared/content 正文渲染和样式 |
| 08 | currency/USD 种子、money 工具与默认币种消费者 | 字典服务、金额字符串类型 | Price 类型及 displayPrice 消费 |
| 09 | 规格/SKU/价格聚合事务、版本、动作权限与预览 | 规格组合/独立改价/上下架/桌面手机预览 | 公共 Product DTO 与共享详情组件 |
| 10 | 公共分页搜索、起价及有效组合过滤 | 查看官网入口及发布对照 | 产品列表、slug 详情、规格选择、SEO/空状态 |
| 11 | 新闻富文本、封面引用、排序/状态/预览 | 新闻编辑、发布/下线及版本冲突提示 | /news 列表详情共享富文本、SEO |
| 12 | banner_slot/banner、在线排期、theme/logo/version | 位置配置、不限条数投放、主题预览 | 三个位置全部 Banner、CSS 变量和移动 Logo |
| 13 | 逐字段热加载、事务/权限/并发及资源泄漏验收 | config.read 状态页、三角色及失败交互 | 英文降级恢复及移动端联调 |
| 14 | 空库 migrate→seed→run、备份恢复和交付说明 | 首次使用引导及菜单授权核对 | 默认主题、空内容、首次部署展示 |

## 必需结构及系统种子

| 结构/种子 | 实施步骤 | 初始化边界 |
| --- | --- | --- |
| 稳定站点身份、存储 root 注册 | 02 | 生成站点/root 标识；DSN 目标身份一致，旧路径持久解析 |
| admin_user、admin_role、admin_permission、admin_role_permission | 03 | 权限目录树 JSON；super_admin/content_admin/viewer；按 code 幂等 |
| admin_session、audit_log | 03 | 建用户归属、scope/auth_version 与审计结构，不种业务记录 |
| 首位超管 | 03 | 默认 admin、必须改初始密码、未绑定；受保护密码或 SEED_ADMIN_PASSWORD；重跑不覆盖已有账户 |
| MFA ticket、pending secret、used timestep、共享限流 | 05 | 建结构，不种密钥/票据；无 HTTP 恢复绕过 |
| media、media_reference | 06 | 创建目标媒体与引用结构，不回填 URL、不种样本文件 |
| currency | 08 | USD/minor_unit=2；重跑不覆盖已存在精度 |
| product/spec/spec_value/sku/sku_value/sku_price | 09 | 创建目标结构，默认 SKU 在真实产品创建时生成；不种演示产品 |
| article | 11 | 净化富文本、封面、排序、时间、version；不种演示新闻 |
| banner_slot/banner、site_config | 12 | 仅 home_hero/products_header/research_header 位置；无张数上限；不种投放 |

步骤 03 数据库种子加载器尚未实现，现有 Python 校验仅检查静态输入。权限 API 精确登记见 IMPLEMENTATION_CONTRACT.md，配置的 13 个现有叶子字段、来源/默认/校验/消费者及场景见 CONFIG_RELOAD_MATRIX.md。

## 编辑器与共享构建验证

选择 Lexical，所有子包锁相同版本；候选 0.52.0 的公开 peerDependencies 支持 React >=18.x、TypeScript >=5.2。隔离目录 /private/tmp/shop-editor-probe 使用 React 19.2.5、TypeScript 6.0.3 编译最小 React 编辑器（正文、标题、引用、错误边界），tsc strict/bundler/react-jsx 检查通过。随后将探针 React/react-dom 更新为仓库实际安装的 19.3.0，再次类型检查通过。步骤 07 接入还必须通过现有 Umi/Next 的实际编辑器构建及浏览器验收，不能以此探针代替节点/净化/媒体选择验收。

共享 contracts 从仓库外层直接导入已通过三端基线类型检查与两端生产构建。shared/content 将放置纯 React 渲染与样式，后台编辑器单独封装；官网只引入展示组件，SKU 选择使用 Client Component。已读取 website/AGENTS.md 和本地 Next use-client 指南；没有提前修改官网正文运行逻辑。

## 交付与后续验收

步骤 01 文档和静态契约登记已完成，编辑器实际 Umi/Next 接入验证属于步骤 07；热加载及安全业务均未实施。提交由用户执行，当前无提交标识。本轮未新增数据库表、运行配置字段、错误处理器、审计 action 或生产权限；错误码和路径是目标登记。

工量区分三类：步骤 02 的 DB/HTTP/storage 多资源热切换是独立复杂任务；步骤 03–05 和 06–12 分别涉及安全并发与内容三端实现；步骤 13–14 必须留出真实 PG、浏览器及恢复演练时间。当前基线仅支持按步骤拆分工作，尚无资源切换原型及真实环境验证数据，不给出未经验证的日历完工日期。

## 步骤 01 八项收尾

1. 做了什么：复核既有静态契约/种子，补齐逐路由权限、公共及预览 DTO 规则、三端任务、系统种子与全量字段登记；同步 README 和本地计划进度。
2. 验证证据：后端 test/vet/build、后台 setup/typecheck/build、官网 build/typecheck、权限校验与负向测试均通过；Lexical React 19.2.5/19.3.0 隔离类型探针通过，未进行浏览器/数据库验收。
3. 联动清单：proto→生成、schema→迁移/生成、构造函数→Wire、repo→fake、新表→defaultTables 均不适用；运行接口及构造函数未改。
4. 新增资产：实施基线文档、路由/错误码目标登记和字段校验/测试场景；没有新增运行错误码、配置、表、Redis key、任务、审计或云依赖。
5. 对称性自查：公共/管理接口分开，整份编辑仍需独立发布/改价授权，匿名/受限 scope 精确登记；实际中间件挂载及反向状态守卫在各实施步骤验证。
6. 遗留项：02–14 业务实施、真实 PostgreSQL、编辑器完整构建与浏览器媒体测试、初始化/恢复演练未执行。
7. commit message：docs(cms): 完成改造基线与三端接口配置登记。按本轮四个文档整理提交，既有未提交代码与配置另行核对；无 protobuf 版本戳清理需求。
8. 结论沉淀：ID 字符串、金额展示、预览、正文协议与编辑器选择落在 IMPLEMENTATION_CONTRACT；配置资源消费者落在 CONFIG_RELOAD_MATRIX；验证边界落在本文。
