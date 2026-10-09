# CMS 接口与部署说明

管理 API 前缀 `/admin/api/v1`；公共内容 API 前缀 `/api/v1`。管理后台通过 `/backend` 代理访问 Go 服务。官网在服务端使用 `BACKEND_URL` 访问公共接口。

| 方法与路径 | 行为 |
|---|---|
| POST /admin/api/v1/login | username、password 登录，返回 HttpOnly 会话 Cookie |
| GET /admin/api/v1/me | 当前管理员 username |
| POST /admin/api/v1/logout | 撤销当前会话 |
| POST /admin/api/v1/password | current、password 改密，撤销所有会话 |
| GET /admin/api/v1/audit | 最近 100 条操作日志 |
| GET /admin/api/v1/media | 最新 200 条素材 |
| POST /admin/api/v1/media | multipart 的 file 字段上传，最大 20MB |
| GET /admin/api/v1/articles | 最新 200 条文章（含草稿、下线） |
| POST /admin/api/v1/articles | 创建文章，id 为 0 |
| PUT /admin/api/v1/articles | 更新文章，id 为已有文章 ID |
| DELETE /admin/api/v1/articles/{id} | 删除指定文章 |
| GET /admin/api/v1/settings | 读取官网配置 |
| PUT /admin/api/v1/settings | 保存整份配置 |
| GET /admin/api/v1/products?q=关键词 | 最新 200 条商品，含草稿与下架，搜索名称、别名、CAS、序列、分类 |
| POST /admin/api/v1/products | 新建商品，id 为 0 |
| PUT /admin/api/v1/products | 更新商品，id 为已有商品 ID；通过 status 上下架 |
| DELETE /admin/api/v1/products/{id} | 删除商品 |
| GET /api/v1/products?q=关键词 | 最新 200 条已上架商品，可搜索 |
| GET /api/v1/products/{slug} | 指定已上架商品；草稿与下架返回 404 |
| GET /api/v1/site | 公共官网配置，无私密管理数据 |
| GET /api/v1/articles | 最新 200 条已发布文章 |
| GET /api/v1/articles/{slug} | 指定已发布文章 |
| GET /uploads/{filename} | 读取公开素材 |

除登录外的管理接口需要有效 `admin_session` Cookie。登录以及所有管理写请求需要 `X-Requested-With: peptide-admin` 请求头；不设置开放跨域策略。管理写入返回错误时不会把数据库报错或连接串交给浏览器。

文章字段：id、title、slug、summary、body、cover、status、seoTitle、seoDescription、updatedAt。正文以纯文本存储和渲染，status 为 draft / published / offline。

商品字段：id、name、slug、category、aliases、cas、formula、molecularWeight、sequence、purity、specifications、storage、solubility、summary、description、images、status、seoTitle、seoDescription、updatedAt。images 为最多 20 个图片地址的数组，第一张为主图，全部属于商品，不绑定颜色。status 为 draft / published / offline；正文为纯文本；不含价格或支付字段。Slug 唯一，小写英文、数字和连字符；q 最长 200 字节。分类在商品表单直接填写，独立分类管理仍待实现。

配置字段：brand、logo、primaryColor、backgroundColor、description、email、whatsapp、disclaimer、banners。Banner 子项为 title、subtitle、image、link；最多 3 项。颜色为 #RRGGBB；URL 仅接受站内绝对路径或 HTTPS。Logo 与 Banner 图片通过素材库引用。

素材上传本地目录必须持久化和备份。MP4/WebM 仅保留原文件，不生成封面、切片或自适应流。所有上传资料公开可访问，不适合私密报告。

当前列表有固定最大条数，尚未实现服务端分页；商品支持搜索，其他列表不支持搜索；素材删除、重命名与分组也未实现。会话、文章、配置和素材元数据存放于 PG；登录限流状态为单进程内存。

初始化命令与密码设置见根目录 README。生产需将官网流量反向代理到 3000，将管理后台静态资源及 `/backend/` 分别代理至前端文件与 Go 服务；后端 8080、PG 5432 不应直接开放公网。上传入口设置至少 21MB 请求上限。

后端配置：默认加载工作目录下 `configs/config.local.yaml`，可用 `-conf` 指定文件，环境变量覆盖 YAML。`-migrate` 按顺序执行内嵌增量迁移并记录版本，可用于升级已有数据库。

配置模板 `backend/configs/config.local.example.yaml` 可提交到 Git；本地文件 `backend/configs/config.local.yaml` 以及其他实际运行 YAML 被 `.gitignore` 排除。复制模板后填写 `data.database.dsn`。分组为 app（服务名）、server.http（监听与超时）、data.database（PG 与连接池）、data.storage（上传目录）、auth（Cookie）。单服务入口 `cmd/server` 调用 `internal/pkg/bootstrap`，统一装配官网与管理接口。
