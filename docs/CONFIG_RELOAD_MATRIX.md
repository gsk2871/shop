# 后端配置热加载字段矩阵

默认值 → YAML → 启动时捕获的环境覆盖。整版校验并准备独立资源后原子发布，失败保留最后可用版本；请求、上传、预览和事务使用同一版本。状态接口仅显示版本、时间、错误与来源，不输出配置值。

全部 37 个有效 YAML 字段如下。DSN、HTTP 地址及目录可热加载。超管初始化只读取显式命令参数和环境变量，币种直接读取 currency 表，均不属于运行 YAML。

| 字段 | 默认值／校验 | 实际消费者 | 验证证据 |
| --- | --- | --- | --- |
| app.name | peptide-site；非空 | 动态日志 app 属性 | TestDynamicLogLevelAndRequestIdentity |
| log.level | info；debug/info/warn/error | slog Enabled | TestDynamicLogLevelAndRequestIdentity |
| reload.interval | 5s；100ms–1h | 下一轮 watcher 定时 | TestLiveTimeoutAndWatcherConsumers |
| server.http.addr | 127.0.0.1:8080；合法 host:port | 先预绑定、再切换、排空旧 listener | TestRuntimePostgres：占用回退、地址切换 |
| server.http.timeout | 30s；正数 | 请求 context 和读写预算 | TestLiveTimeoutAndWatcherConsumers |
| server.http.read_header_timeout | 10s；正数 | 新连接及 keepalive 请求头 deadline | 同上，两个慢请求头场景 |
| server.http.read_timeout | 60s；正数 | 请求体 deadline | 同上，慢请求体返回 408 |
| server.http.write_timeout | 90s；大于 timeout/read_timeout | 响应写入 deadline，与请求预算取较小值 | 同上，延迟响应失败 |
| server.http.idle_timeout | 120s；正数 | 空闲连接 deadline | 同上，实际连接断开 |
| data.database.dsn | 必填；环境 DATABASE_DSN 可覆盖 | 新池 Ping，完整迁移、列／约束／索引及站点身份核对 | TestRuntimePostgres；TestReloadResourceConsumers 的 application_name |
| data.database.max_open_conns | 10；>0 | 新 sql.DB 池上限 | TestReloadResourceConsumers 的 Stats |
| data.database.max_idle_conns | 5；0≤值≤max_open | 新池空闲上限 | 同上，真实池与连接寿命 |
| data.database.conn_max_lifetime | 30m；正数 | 新池连接回收 | 同上，MaxLifetimeClosed 实际增加 |
| data.database.conn_max_idle_time | 15m；正数 | 新连接池空闲连接回收 | TestDatabaseTimeoutsAndPool |
| data.database.connect_timeout | 10s；1s–30s | pgx 建连超时，优先于 DSN | TestDatabaseTimeoutsAndPool |
| data.database.statement_timeout | 5s；1ms–30s | 每条连接启动参数，SQL 执行超时 | TestDatabaseTimeoutsAndPool 的 pg_sleep 超时 |
| data.database.lock_timeout | 3s；≥1ms，且小于 statement_timeout | 每条连接启动参数，SQL 等锁超时 | TestDatabaseTimeoutsAndPool 的锁竞争 |
| data.database.log_sql | false；需同时 log.level=debug | GORM SQL 日志，参数不输出 | TestSQLLoggingPolicyAndRedaction |
| data.database.slow_sql_ms | 200；1–30000 | 慢查询 warn 日志阈值 | TestSQLLoggingPolicyAndRedaction |
| data.storage.upload_dir | ./storage/uploads；可写目录 | 上传路径和持久 root 注册 | TestRuntimePostgres：切换、旧文件和 Range；配套恢复演练 |
| data.storage.max_image_bytes | 20 MiB；>0，≤1 TiB | 图片流写入限制／策略接口 | TestStreamingUploadPolicyAndReferences；TestReloadResourceConsumers |
| data.storage.max_video_bytes | 200 MiB；同上 | 视频流写入限制／策略接口 | 分类限额拒绝；TestReloadDrainsUploadAndTransaction：在途视频保留旧策略；真实 MP4 官网 readyState=4 |
| data.storage.max_document_bytes | 历史兼容字段 | 不影响有效上传策略，PDF 始终拒绝 | TestPDFRejectedEvenByLegacyPolicy |
| data.storage.allowed_types | JPEG/PNG/WebP/GIF/MP4/WebM；无重复；历史 application/pdf 可解析但不生效 | 上传策略接口及内容嗅探仅允许配置中的图片、视频类型，始终排除 PDF | PNG 被更新策略拒绝、SVG 拒绝；TestPDFRejectedEvenByLegacyPolicy |
| data.storage.cleanup_interval | 1m；100ms–1h | 新版本后台任务、旧任务取消 | TestReloadResourceConsumers；删除和崩溃临时文件测试 |
| auth.cookie_secure | true；环境 COOKIE_SECURE 可覆盖 | 新签发／注销 Cookie | TestReloadResourceConsumers；HttpOnly/SameSite 测试 |
| auth.session_ttl | 8h；>0，≤24h | 新完整会话 DB expiry 和 MaxAge | TestReloadResourceConsumers：3600s |
| auth.onboarding_ttl | 15m；>0，≤1h | 新受限会话、待绑定 expiry | 同上：540s，首次设置 HTTP 流程 |
| auth.mfa_ticket_ttl | 5m；>0，≤15m | 新登录票据 expiry | 同上：120s 和数据库 expiry |
| auth.mfa_max_attempts | 5；1–20 | 票据消费事务、失败次数 | 同上，2 次失败后正确验证码仍拒绝 |
| auth.login_limit | 5；≥1；按一分钟窗口 | PostgreSQL 共享计数 | 同上；TestSeedAndTwoPhaseLogin 跨仓储共享 |
| auth.trusted_proxies | 空；合法 CIDR | 来源 IP 与 HTTPS Origin 判断 | TestTrustedProxyConsumers：可信／不可信两侧 |
| auth.totp.enabled | true；可配置为 true/false | 登录验证码、强制绑定、密码再认证与 me.totpEnabled；新请求使用当前配置，重新开启后密码登录会话需重登 | TestTOTPEnabledConfiguration；TestTOTPDisabledLoginBindingAndReenable；TestReloadTOTPEnabledConsumer |
| auth.totp.issuer | Research Supply Admin；非空 | 新绑定 URI 和本地二维码 | TestReloadResourceConsumers |
| auth.totp.active_key_id | 运行时必填；存在于 keys | 新绑定 AES-GCM 加密 | 同上，pending.key_id 实际更新 |
| auth.totp.keys | 运行时必填；每项 base64 32 字节 | 当前／历史密文解密；在途旧版本保护 | TestSeedAndTwoPhaseLogin；TestKeyRemovalWaitsForOldLease；恢复启动检查 |
| contact.geoip_database_path | 空；非空路径需可读有效 MMDB | 新资源打开国家 Reader，旧请求排空后关闭，失败保留当前配置 | TestCountryDatabase；TestCountryDatabaseReload |

来源接口 `/admin/api/v1/config/status` 需要 config.read。后台每 10 秒刷新状态及上传策略。环境覆盖固定捕获，不重读后来修改的进程环境；被覆盖 YAML 变化可以校验但不能替代环境值，等效配置不增加版本。敏感字段的状态只显示字段名和来源。

Prepare：校验配置、打开独立池、Ping 和结构／身份检查、校验字典及历史密钥、检查目录、新监听预绑定。Commit：持久化目录注册、启动新任务、发布快照和监听。Rollback：释放候选；Drain：等待旧请求结束、取消旧任务、关闭旧池和监听。并发 reload 串行。

已验证配置文件消失及恢复、无效 YAML、错误数据库身份、连接池非法组合、端口占用和密钥使用中的整版拒绝；环境快照、原子文件替换、在途租约排空、重复版本切换、真实读写超时、清理任务及恢复文件均有可复跑测试。TestReloadDrainsUploadAndTransaction 使用真实 PostgreSQL 事务和暂停的视频流，同时切换 DSN、上传目录与视频限额；旧请求完成后才关闭旧池，新上传立即采用新限额。不可读目录未作为单独端到端案例记成功。

外部 Nginx、后台构建环境与官网 BACKEND_URL 不属于后端 YAML，使用各自部署机制。schema 迁移及 seed 均为显式维护命令。

超管用户名由 -seed-admin-username 指定（默认 admin），首次密码由 SEED_ADMIN_PASSWORD 提供；重复 seed 保留已有账号凭据。日志输出为标准输出文本，SQL 绑定参数和数据库错误详情不输出。
