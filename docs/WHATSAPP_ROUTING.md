# WhatsApp 国家客服分配

管理后台通过 `/whatsapp` 管理国家配置，一个国家只归属一条未删除配置，每条配置包含多个 WhatsApp 账号。展开账号列表显示名称、号码、状态及累计分配次数，可直接启停，不显示二维码。账号编辑必须选择二维码图片，复用媒体上传、预览及引用保护。

全站只有一个默认配置，从现有国家配置中选择。设置前提示目标、当前默认及替换影响，取消不提交；确认成功后更新唯一标记。已可用默认不能停用、删除或移除最后可用账号。种子创建 US「美国客服」普通配置并在无默认时选中它，不创建虚构号码或二维码；初始化空账号显示待配置，官网返回 WHATSAPP_UNAVAILABLE。运营补齐真实号码和二维码后可用。重复 seed 保留现有默认和运营数据，补齐超级管理员缺失权限，不变更普通角色已有授权。

## 分配

点击客服按钮时读取账号。可信入口覆盖 `X-Client-IP`，官网服务转发规范化地址；后端先检查调用方属于 `auth.trusted_proxies`，只从专用请求头获取访客 IP。缺失／非法地址返回 400，非可信来源返回 403。国家识别使用本地 MMDB，不进行实时第三方查询；未知／未配置国家、停用配置或空账号池使用默认。

有效池包含启用、未删除且具备有效号码和可用图片媒体的账号。按稳定 ID 升序轮询，寻找大于上次 ID 的第一个可用账号，没有则从最小 ID 开始。配置独立维护 cursor，默认直接匹配和兜底共用该 cursor。4→6 保留原 ID 与位置，新增两账号追加；最后 D 后为 E、F、A、B、C、D，不追平历史量。停用跳过，重新启用回到原 ID 位置。

GET 联系方式在后台推进内部轮询位置和计数。事务采用设置行共享锁、最终组 cursor 排他锁，管理操作先排他锁同一设置行；多实例同组串行，不同组可并发。失败整次回滚。成功提交但响应丢失仍可能计数，重复读取可返回另一账号。统计不代表实际访客、聊天或成交。

## 接口

ID 为十进制字符串，时间为 UTC RFC 3339，保留数据库微秒精度。所有管理写请求复用完整会话、权限、同源保护和审计。

| 方法与后端路径 | 权限／作用 |
| --- | --- |
| GET /admin/api/v1/whatsapp-groups | whatsapp.read；分页、名称或国家代码搜索，返回默认状态 |
| GET /admin/api/v1/whatsapp-groups/{id} | whatsapp.read；配置与完整账号列表 |
| GET /admin/api/v1/whatsapp-countries | whatsapp.read；国家选择范围 |
| POST /admin/api/v1/whatsapp-groups | whatsapp.create；创建配置及账号 |
| PUT /admin/api/v1/whatsapp-groups/{id} | whatsapp.update；聚合编辑 |
| PATCH /admin/api/v1/whatsapp-groups/{id}/enabled | whatsapp.update；配置启停 |
| PATCH /admin/api/v1/whatsapp-groups/{id}/accounts/{accountId}/enabled | whatsapp.update；账号启停 |
| DELETE /admin/api/v1/whatsapp-groups/{id} | whatsapp.delete；软删除配置和子记录 |
| PUT /admin/api/v1/whatsapp-default | whatsapp.default；切换唯一默认 |
| GET /api/v1/whatsapp-status | 公开读取可用状态，不消耗轮询 |
| GET /api/v1/whatsapp-contact | 可信官网代理读取选中账号及二维码 |

官网浏览器调用 `GET /api/contact/whatsapp` 同源代理，无 body、业务 query、点击标识或产品提交。前后端使用 no-store，禁预取和自动重试，用户主动重试作为新分配；HEAD 明确拒绝，不能消耗位置。响应为 accountName、phoneNumber、qrCodeUrl、url。产品文案在浏览器基于已展示名称编码到 wa.me 链接。官网没有联系页、邮件入口或联系表单，旧 `/contact` 重定向到首页。

编辑输入为 name、countryCodes、enabled、accounts；账号为已有 id（新账号省略）、name、phoneNumber、qrCodeMediaId、enabled。已有账号缺席表示软删除，保留账号 ID 和计数。更新、启停、删除传 `expectedUpdatedAt`，指所属配置更新时间，账号操作也同步推进配置时间。默认切换传 groupId、expectedUpdatedAt（全局设置时间）、expectedGroupUpdatedAt（目标配置时间）。数据库时间不一致返回 409 UPDATE_CONFLICT，整次不更新，前端刷新后人工核对。时间原样回传，不通过 JavaScript Date 截断。

其余业务错误包括 COUNTRY_ALREADY_ASSIGNED、DUPLICATE_WHATSAPP_ACCOUNT、DEFAULT_GROUP_PROTECTED、DEFAULT_GROUP_UNAVAILABLE、WHATSAPP_UNAVAILABLE；使用现有 `{code,message,requestId}` 格式。

## 数据与部署

迁移为 `backend/internal/data/migration/000012_whatsapp.up.sql` 及配对 down。五表：whatsapp_group、whatsapp_group_country、whatsapp_account、whatsapp_routing_setting、whatsapp_cursor；业务软删除，无物理外键。二维码逻辑指向 media，media_reference 的 owner_type 为 whatsapp_account，field 为 qr_code。没有访客资料或短期分配结果表，不持久化访客 IP。媒体图片不因列表省略缩略图而移除。

执行后端 `-migrate` 再 `-seed`，普通已有运营角色由超级管理员授权新权限。后端 Go 保持 1.25，MMDB Reader 固定使用 maxminddb-golang/v2 v2.0.0；参考[库文档](https://github.com/oschwald/maxminddb-golang/tree/v2.0.0)。

```yaml
contact:
  geoip_database_path: /persistent/geoip/dbip-country-lite-2026-10.mmdb
```

空路径让合法请求使用默认。非空资源在新配置版本准备阶段打开并验证，失败保留当前资源，旧请求排空后关闭旧 Reader。更新时部署不可变版本文件并修改 YAML 路径；不能原地修改打开的数据库。MMDB 位于私有存储目录，不进 Git。

使用 [DB-IP Country Lite](https://db-ip.com/db/download/ip-to-country-lite) 及其[格式说明](https://db-ip.com/db/format/ip-to-country-lite/mmdb.html)，按其许可在官网页脚显示 IP Geolocation by DB-IP 回链署名。免费库的国家结果不代表访客实际所在地。WhatsApp 链接见[官方帮助](https://faq.whatsapp.com/5913398998672934)。

生产官网入口必须覆盖浏览器提供的 IP header。无上游负载均衡时示例：

```nginx
location / {
    proxy_set_header X-Client-IP $remote_addr;
    proxy_set_header Host $http_host;
    proxy_pass http://127.0.0.1:3000;
}
```

有上游代理时先按可信来源规则还原真实地址。Next 和 Go 仅内网可达，后端 trusted_proxies 只允许实际官网地址，并配置入口限流。

本地 `cd website && pnpm dev` 启动 3000 入口和 3001 私有 Next 服务，入口从连接来源覆盖请求头，访客应访问 3000。本地 loopback 识别国家未知，因此使用默认配置；不能用浏览器伪造 header 选择国家。`pnpm start` 的生产 Next 服务仍需可信入口代理。国家库、私有 YAML 和账号图片单独备份。

## 页面与验证

H5 覆盖导航、产品列表和详情、页脚、客服弹窗。手机优先直接打开 WhatsApp，二维码完整等比展示；弹层限制可视高度并内部滚动、背景滚动锁定、关闭恢复焦点及位置。固定入口留安全区域，窄屏重排内容，触控目标至少 44 CSS 像素。

验证命令：`go test ./...`，新模块 `go test -race`，两个前端 typecheck／build，权限种子 Python 校验。真实数据库测试使用独立 PostgreSQL，设置 TEST_DATABASE_DSN；国家测试可设置 TEST_GEOIP_PATH 到固定版本文件。验证覆盖并发均分、4→6、启停、更新时间冲突、默认唯一、默认保护、媒体引用、种子保留和失败回滚，以及可信 header、IPv4／IPv6、公共 DTO、状态读取不计数和无缓存。

浏览器验收应覆盖手机断点、横竖屏、客服加载／失败／关闭、直接聊天及扫码。真实账号未配置时只能验收暂不可用提示；实际扫码须使用运营提供的真实账号二维码。
