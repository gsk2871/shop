-- 全新站点的完整初始结构；逻辑引用由业务事务维护。

-- 站点身份
CREATE TABLE site_identity (
  id bigint NOT NULL,
  site_id text NOT NULL DEFAULT (gen_random_uuid())::text,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT site_identity_id_check CHECK ((id = 1)),
  CONSTRAINT site_identity_pkey PRIMARY KEY (id)
);
COMMENT ON TABLE site_identity IS '站点身份';
COMMENT ON COLUMN site_identity.id IS '站点身份单例标识';
COMMENT ON COLUMN site_identity.site_id IS '站点标识';
COMMENT ON COLUMN site_identity.created_at IS '创建时间';

-- 本地媒体存储根目录
CREATE TABLE storage_root (
  id text NOT NULL,
  path text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT storage_root_path_key UNIQUE (path),
  CONSTRAINT storage_root_pkey PRIMARY KEY (id)
);
COMMENT ON TABLE storage_root IS '本地媒体存储根目录';
COMMENT ON COLUMN storage_root.id IS '存储根标识';
COMMENT ON COLUMN storage_root.path IS '存储目录';
COMMENT ON COLUMN storage_root.created_at IS '创建时间';

-- 后台角色
CREATE TABLE admin_role (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  code text NOT NULL,
  name text NOT NULL,
  is_super boolean NOT NULL DEFAULT false,
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  created_by bigint,
  updated_by bigint,
  CONSTRAINT admin_role_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX admin_role_code ON admin_role USING btree (code) WHERE (deleted_at IS NULL);
COMMENT ON TABLE admin_role IS '后台角色';
COMMENT ON COLUMN admin_role.id IS '后台角色标识';
COMMENT ON COLUMN admin_role.code IS '编码';
COMMENT ON COLUMN admin_role.name IS '名称';
COMMENT ON COLUMN admin_role.is_super IS '是否超级管理员角色';
COMMENT ON COLUMN admin_role.enabled IS '是否启用';
COMMENT ON COLUMN admin_role.created_at IS '创建时间';
COMMENT ON COLUMN admin_role.updated_at IS '更新时间';
COMMENT ON COLUMN admin_role.deleted_at IS '删除时间';
COMMENT ON COLUMN admin_role.created_by IS '创建人';
COMMENT ON COLUMN admin_role.updated_by IS '更新人';

-- 后台权限节点
CREATE TABLE admin_permission (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  code text NOT NULL,
  parent_id bigint,
  name text NOT NULL,
  node_type text NOT NULL,
  route text,
  permission_code text,
  sort integer NOT NULL DEFAULT 0,
  visibility text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT admin_permission_code_key UNIQUE (code),
  CONSTRAINT admin_permission_node_type_check CHECK ((node_type = ANY (ARRAY['directory'::text, 'menu'::text, 'action'::text]))),
  CONSTRAINT admin_permission_permission_code_key UNIQUE (permission_code),
  CONSTRAINT admin_permission_pkey PRIMARY KEY (id)
);
CREATE INDEX admin_permission_parent ON admin_permission USING btree (parent_id, sort);
COMMENT ON TABLE admin_permission IS '后台权限节点';
COMMENT ON COLUMN admin_permission.id IS '后台权限节点标识';
COMMENT ON COLUMN admin_permission.code IS '编码';
COMMENT ON COLUMN admin_permission.parent_id IS '逻辑指向 admin_permission.id';
COMMENT ON COLUMN admin_permission.name IS '名称';
COMMENT ON COLUMN admin_permission.node_type IS '节点类型:directory=目录 menu=菜单 action=动作';
COMMENT ON COLUMN admin_permission.route IS '后台路由';
COMMENT ON COLUMN admin_permission.permission_code IS '权限标识';
COMMENT ON COLUMN admin_permission.sort IS '展示顺序';
COMMENT ON COLUMN admin_permission.visibility IS '可见性:空=常规权限 authenticated_self_service=已登录用户自助访问';
COMMENT ON COLUMN admin_permission.created_at IS '创建时间';
COMMENT ON COLUMN admin_permission.updated_at IS '更新时间';

-- 角色权限关联
CREATE TABLE admin_role_permission (
  role_id bigint NOT NULL,
  permission_id bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT admin_role_permission_pkey PRIMARY KEY (role_id, permission_id)
);
COMMENT ON TABLE admin_role_permission IS '角色权限关联';
COMMENT ON COLUMN admin_role_permission.role_id IS '逻辑指向 admin_role.id';
COMMENT ON COLUMN admin_role_permission.permission_id IS '逻辑指向 admin_permission.id';
COMMENT ON COLUMN admin_role_permission.created_at IS '创建时间';

-- 后台用户
CREATE TABLE admin_user (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  username text NOT NULL,
  display_name text NOT NULL,
  password_hash text NOT NULL,
  role_id bigint NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  must_change_pwd boolean NOT NULL DEFAULT true,
  auth_version bigint NOT NULL DEFAULT 1,
  totp_key_id text,
  totp_cipher text,
  last_login_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  created_by bigint,
  updated_by bigint,
  CONSTRAINT admin_user_pkey PRIMARY KEY (id)
);
CREATE INDEX admin_user_role ON admin_user USING btree (role_id) WHERE (deleted_at IS NULL);
CREATE UNIQUE INDEX admin_user_username ON admin_user USING btree (username) WHERE (deleted_at IS NULL);
COMMENT ON TABLE admin_user IS '后台用户';
COMMENT ON COLUMN admin_user.id IS '后台用户标识';
COMMENT ON COLUMN admin_user.username IS '登录名';
COMMENT ON COLUMN admin_user.display_name IS '显示名称';
COMMENT ON COLUMN admin_user.password_hash IS '密码摘要';
COMMENT ON COLUMN admin_user.role_id IS '逻辑指向 admin_role.id';
COMMENT ON COLUMN admin_user.enabled IS '是否启用';
COMMENT ON COLUMN admin_user.must_change_pwd IS '是否需要修改密码';
COMMENT ON COLUMN admin_user.auth_version IS '认证版本';
COMMENT ON COLUMN admin_user.totp_key_id IS '密钥标识';
COMMENT ON COLUMN admin_user.totp_cipher IS '验证器密文';
COMMENT ON COLUMN admin_user.last_login_at IS '最近登录时间';
COMMENT ON COLUMN admin_user.created_at IS '创建时间';
COMMENT ON COLUMN admin_user.updated_at IS '更新时间';
COMMENT ON COLUMN admin_user.deleted_at IS '删除时间';
COMMENT ON COLUMN admin_user.created_by IS '创建人';
COMMENT ON COLUMN admin_user.updated_by IS '更新人';

-- 后台登录会话
CREATE TABLE admin_session (
  token_hash varchar(64) NOT NULL,
  expires_at timestamptz NOT NULL,
  user_id bigint NOT NULL,
  scope text NOT NULL,
  auth_version bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  verified_at timestamptz,
  CONSTRAINT admin_session_pkey PRIMARY KEY (token_hash),
  CONSTRAINT admin_session_scope_check CHECK ((scope = ANY (ARRAY['full'::text, 'onboarding'::text, 'password_change'::text])))
);
CREATE INDEX admin_session_expiry ON admin_session USING btree (expires_at);
CREATE INDEX admin_session_user ON admin_session USING btree (user_id);
COMMENT ON TABLE admin_session IS '后台登录会话';
COMMENT ON COLUMN admin_session.token_hash IS '令牌摘要';
COMMENT ON COLUMN admin_session.expires_at IS '失效时间';
COMMENT ON COLUMN admin_session.user_id IS '逻辑指向 admin_user.id';
COMMENT ON COLUMN admin_session.scope IS '会话范围:full=完整 onboarding=首次设置 password_change=修改密码';
COMMENT ON COLUMN admin_session.auth_version IS '认证版本';
COMMENT ON COLUMN admin_session.created_at IS '创建时间';
COMMENT ON COLUMN admin_session.verified_at IS '最近敏感操作验证时间';

-- 后台登录票据
CREATE TABLE admin_mfa_ticket (
  token_hash text NOT NULL,
  user_id bigint NOT NULL,
  auth_version bigint NOT NULL,
  expires_at timestamptz NOT NULL,
  attempts integer NOT NULL DEFAULT 0,
  consumed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT admin_mfa_ticket_pkey PRIMARY KEY (token_hash)
);
CREATE INDEX admin_mfa_ticket_expiry ON admin_mfa_ticket USING btree (expires_at);
CREATE INDEX admin_mfa_ticket_user ON admin_mfa_ticket USING btree (user_id);
COMMENT ON TABLE admin_mfa_ticket IS '后台登录票据';
COMMENT ON COLUMN admin_mfa_ticket.token_hash IS '令牌摘要';
COMMENT ON COLUMN admin_mfa_ticket.user_id IS '逻辑指向 admin_user.id';
COMMENT ON COLUMN admin_mfa_ticket.auth_version IS '认证版本';
COMMENT ON COLUMN admin_mfa_ticket.expires_at IS '失效时间';
COMMENT ON COLUMN admin_mfa_ticket.attempts IS '尝试次数';
COMMENT ON COLUMN admin_mfa_ticket.consumed_at IS '票据使用时间';
COMMENT ON COLUMN admin_mfa_ticket.created_at IS '创建时间';

-- 待绑定验证器
CREATE TABLE admin_totp_pending (
  user_id bigint NOT NULL,
  auth_version bigint NOT NULL,
  key_id text NOT NULL,
  cipher text NOT NULL,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT admin_totp_pending_pkey PRIMARY KEY (user_id)
);
COMMENT ON TABLE admin_totp_pending IS '待绑定验证器';
COMMENT ON COLUMN admin_totp_pending.user_id IS '逻辑指向 admin_user.id';
COMMENT ON COLUMN admin_totp_pending.auth_version IS '认证版本';
COMMENT ON COLUMN admin_totp_pending.key_id IS '加密密钥标识';
COMMENT ON COLUMN admin_totp_pending.cipher IS '验证器密文';
COMMENT ON COLUMN admin_totp_pending.expires_at IS '失效时间';
COMMENT ON COLUMN admin_totp_pending.created_at IS '创建时间';

-- 已使用验证时间步
CREATE TABLE admin_totp_used (
  user_id bigint NOT NULL,
  timestep bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT admin_totp_used_pkey PRIMARY KEY (user_id, timestep)
);
CREATE INDEX admin_totp_used_time ON admin_totp_used USING btree (created_at);
COMMENT ON TABLE admin_totp_used IS '已使用验证时间步';
COMMENT ON COLUMN admin_totp_used.user_id IS '逻辑指向 admin_user.id';
COMMENT ON COLUMN admin_totp_used.timestep IS '验证时间步';
COMMENT ON COLUMN admin_totp_used.created_at IS '创建时间';

-- 后台安全限流
CREATE TABLE admin_rate_limit (
  key_hash text NOT NULL,
  window_start timestamptz NOT NULL,
  attempts integer NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT admin_rate_limit_pkey PRIMARY KEY (key_hash)
);
CREATE INDEX admin_rate_limit_time ON admin_rate_limit USING btree (updated_at);
COMMENT ON TABLE admin_rate_limit IS '后台安全限流';
COMMENT ON COLUMN admin_rate_limit.key_hash IS '限流标识摘要';
COMMENT ON COLUMN admin_rate_limit.window_start IS '计数窗口开始时间';
COMMENT ON COLUMN admin_rate_limit.attempts IS '尝试次数';
COMMENT ON COLUMN admin_rate_limit.updated_at IS '更新时间';

-- 操作审计日志
CREATE TABLE audit_log (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  action text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  user_id bigint,
  target_type text NOT NULL DEFAULT ''::text,
  target_id text NOT NULL DEFAULT ''::text,
  result text NOT NULL DEFAULT 'success'::text,
  request_id text NOT NULL DEFAULT ''::text,
  ip text NOT NULL DEFAULT ''::text,
  changes jsonb NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT audit_log_pkey PRIMARY KEY (id),
  CONSTRAINT audit_log_result_check CHECK ((result = ANY (ARRAY['success'::text, 'failure'::text])))
);
CREATE INDEX audit_log_actor_time ON audit_log USING btree (user_id, created_at DESC);
COMMENT ON TABLE audit_log IS '操作审计日志';
COMMENT ON COLUMN audit_log.id IS '操作审计日志标识';
COMMENT ON COLUMN audit_log.action IS '操作动作';
COMMENT ON COLUMN audit_log.created_at IS '创建时间';
COMMENT ON COLUMN audit_log.user_id IS '逻辑指向 admin_user.id';
COMMENT ON COLUMN audit_log.target_type IS '操作对象类型:auth=认证 user=用户 role=角色 media=素材 product=商品 product_category=商品分类 article=新闻 banner=Banner site_config=品牌配置 whatsapp_group=客服配置 faq=问答';
COMMENT ON COLUMN audit_log.target_id IS '操作对象标识';
COMMENT ON COLUMN audit_log.result IS '结果:success=成功 failure=失败';
COMMENT ON COLUMN audit_log.request_id IS '请求标识';
COMMENT ON COLUMN audit_log.ip IS '请求来源 IP';
COMMENT ON COLUMN audit_log.changes IS '变更内容';

-- 图片目录
CREATE TABLE media_group (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  name text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT media_group_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX media_group_name ON media_group USING btree (lower(name)) WHERE (deleted_at IS NULL);
COMMENT ON TABLE media_group IS '图片目录';
COMMENT ON COLUMN media_group.id IS '图片目录标识';
COMMENT ON COLUMN media_group.name IS '目录名称';
COMMENT ON COLUMN media_group.created_at IS '创建时间';
COMMENT ON COLUMN media_group.updated_at IS '更新时间';
COMMENT ON COLUMN media_group.deleted_at IS '删除时间';

-- COS 图片上传批次
CREATE TABLE media_upload_batch (
  id text NOT NULL,
  uploader_id bigint NOT NULL,
  bucket text NOT NULL,
  region text NOT NULL,
  public_domain text NOT NULL,
  key_prefix text NOT NULL,
  max_bytes bigint NOT NULL,
  expires_at timestamptz NOT NULL,
  cleanup_after timestamptz NOT NULL,
  cleaned_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT media_upload_batch_max_bytes_check CHECK ((max_bytes > 0)),
  CONSTRAINT media_upload_batch_pkey PRIMARY KEY (id)
);
CREATE INDEX media_upload_cleanup ON media_upload_batch USING btree (cleanup_after) WHERE (cleaned_at IS NULL);
COMMENT ON TABLE media_upload_batch IS 'COS 图片上传批次';
COMMENT ON COLUMN media_upload_batch.id IS '上传批次标识';
COMMENT ON COLUMN media_upload_batch.uploader_id IS '上传者';
COMMENT ON COLUMN media_upload_batch.bucket IS '存储桶';
COMMENT ON COLUMN media_upload_batch.region IS '存储地域';
COMMENT ON COLUMN media_upload_batch.public_domain IS '图片访问域名';
COMMENT ON COLUMN media_upload_batch.key_prefix IS '对象路径前缀';
COMMENT ON COLUMN media_upload_batch.max_bytes IS '图片大小上限（字节）';
COMMENT ON COLUMN media_upload_batch.expires_at IS '批次失效时间';
COMMENT ON COLUMN media_upload_batch.cleanup_after IS '允许清理时间';
COMMENT ON COLUMN media_upload_batch.cleaned_at IS '临时文件清理完成时间';
COMMENT ON COLUMN media_upload_batch.created_at IS '创建时间';

-- 媒体素材
CREATE TABLE media (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  name text NOT NULL,
  url text NOT NULL,
  mime text NOT NULL,
  size bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  storage_root_id text NOT NULL,
  storage_key text NOT NULL,
  media_type text NOT NULL,
  width integer,
  height integer,
  available boolean NOT NULL DEFAULT true,
  cleanup_pending boolean NOT NULL DEFAULT false,
  uploaded_by bigint,
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  group_id bigint,
  storage_provider text NOT NULL DEFAULT 'local'::text,
  status text NOT NULL DEFAULT 'ready'::text,
  batch_id text,
  temporary_cleaned boolean NOT NULL DEFAULT false,
  CONSTRAINT media_media_type_check CHECK ((media_type = ANY (ARRAY['image'::text, 'video'::text, 'document'::text]))),
  CONSTRAINT media_pkey PRIMARY KEY (id),
  CONSTRAINT media_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'ready'::text, 'failed'::text, 'canceled'::text]))),
  CONSTRAINT media_storage_provider_check CHECK ((storage_provider = ANY (ARRAY['local'::text, 'cos'::text]))),
  CONSTRAINT media_size_check CHECK (size > 0),
  CONSTRAINT media_dimensions_check CHECK ((width IS NULL OR width > 0) AND (height IS NULL OR height > 0)),
  CONSTRAINT media_cos_batch_check CHECK (storage_provider <> 'cos' OR batch_id IS NOT NULL),
  CONSTRAINT media_ready_url_check CHECK (status <> 'ready' OR btrim(url) <> '')
);
CREATE INDEX media_available_type ON media USING btree (media_type, id DESC) WHERE (available AND (deleted_at IS NULL));
CREATE INDEX media_batch ON media USING btree (batch_id) WHERE (batch_id IS NOT NULL);
CREATE INDEX media_group_list ON media USING btree (group_id, id DESC) WHERE (available AND (deleted_at IS NULL) AND (media_type = 'image'::text));
CREATE UNIQUE INDEX media_storage_key ON media USING btree (storage_root_id, storage_key);
COMMENT ON TABLE media IS '媒体素材';
COMMENT ON COLUMN media.id IS '媒体素材标识';
COMMENT ON COLUMN media.name IS '名称';
COMMENT ON COLUMN media.url IS '媒体访问地址';
COMMENT ON COLUMN media.mime IS '媒体 MIME 类型';
COMMENT ON COLUMN media.size IS '文件大小（字节）';
COMMENT ON COLUMN media.created_at IS '创建时间';
COMMENT ON COLUMN media.storage_root_id IS '存储位置标识:local=本地存储根标识 cos=地域与存储桶标识';
COMMENT ON COLUMN media.storage_key IS '存储对象标识';
COMMENT ON COLUMN media.media_type IS '媒体类型:image=图片 video=视频 document=文档';
COMMENT ON COLUMN media.width IS '宽度（像素）';
COMMENT ON COLUMN media.height IS '高度（像素）';
COMMENT ON COLUMN media.available IS '素材可用状态';
COMMENT ON COLUMN media.cleanup_pending IS '等待文件清理';
COMMENT ON COLUMN media.uploaded_by IS '上传者';
COMMENT ON COLUMN media.updated_at IS '更新时间';
COMMENT ON COLUMN media.deleted_at IS '删除时间';
COMMENT ON COLUMN media.group_id IS '逻辑指向 media_group.id，空值表示未分组';
COMMENT ON COLUMN media.storage_provider IS '存储方式:local=本地 cos=腾讯云对象存储';
COMMENT ON COLUMN media.status IS '上传状态:pending=待完成 ready=可用 failed=失败 canceled=已取消';
COMMENT ON COLUMN media.batch_id IS '逻辑指向 media_upload_batch.id';
COMMENT ON COLUMN media.temporary_cleaned IS '临时图片清理完成状态';

-- 素材引用
CREATE TABLE media_reference (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  media_id bigint NOT NULL,
  owner_type text NOT NULL,
  owner_id text NOT NULL,
  field text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT media_reference_media_id_owner_type_owner_id_field_key UNIQUE (media_id, owner_type, owner_id, field),
  CONSTRAINT media_reference_pkey PRIMARY KEY (id)
);
CREATE INDEX media_reference_owner ON media_reference USING btree (owner_type, owner_id);
COMMENT ON TABLE media_reference IS '素材引用';
COMMENT ON COLUMN media_reference.id IS '素材引用标识';
COMMENT ON COLUMN media_reference.media_id IS '逻辑指向 media.id';
COMMENT ON COLUMN media_reference.owner_type IS '引用对象类型:product=商品 article=文章 banner=Banner site_config=品牌配置 whatsapp_account=客服账号';
COMMENT ON COLUMN media_reference.owner_id IS '引用对象标识';
COMMENT ON COLUMN media_reference.field IS '引用字段:detail=商品正文 image=图片 report=检测报告 body=文章正文 cover=文章封面 mobile_image=手机图片 logo=品牌图片 qr_code=客服二维码';
COMMENT ON COLUMN media_reference.created_at IS '创建时间';

-- 币种字典
CREATE TABLE currency (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  code text NOT NULL,
  minor_unit integer NOT NULL,
  symbol text NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT currency_code_key UNIQUE (code),
  CONSTRAINT currency_minor_unit_check CHECK (((minor_unit >= 0) AND (minor_unit <= 6))),
  CONSTRAINT currency_pkey PRIMARY KEY (id)
);
COMMENT ON TABLE currency IS '币种字典';
COMMENT ON COLUMN currency.id IS '币种字典标识';
COMMENT ON COLUMN currency.code IS '编码';
COMMENT ON COLUMN currency.minor_unit IS '最小货币单位精度';
COMMENT ON COLUMN currency.symbol IS '货币符号';
COMMENT ON COLUMN currency.enabled IS '是否启用';
COMMENT ON COLUMN currency.created_at IS '创建时间';
COMMENT ON COLUMN currency.updated_at IS '更新时间';

-- 商品分类
CREATE TABLE product_category (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  name text NOT NULL,
  sort integer NOT NULL DEFAULT 0,
  version bigint NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  created_by bigint,
  updated_by bigint,
  CONSTRAINT product_category_name_check CHECK ((btrim(name) <> ''::text)),
  CONSTRAINT product_category_pkey PRIMARY KEY (id),
  CONSTRAINT product_category_version_check CHECK ((version > 0))
);
CREATE UNIQUE INDEX product_category_name_active ON product_category USING btree (lower(name)) WHERE (deleted_at IS NULL);
COMMENT ON TABLE product_category IS '商品分类';
COMMENT ON COLUMN product_category.id IS '商品分类标识';
COMMENT ON COLUMN product_category.name IS '分类名称';
COMMENT ON COLUMN product_category.sort IS '展示顺序';
COMMENT ON COLUMN product_category.version IS '内容版本';
COMMENT ON COLUMN product_category.created_at IS '创建时间';
COMMENT ON COLUMN product_category.updated_at IS '更新时间';
COMMENT ON COLUMN product_category.deleted_at IS '删除时间';
COMMENT ON COLUMN product_category.created_by IS '创建人';
COMMENT ON COLUMN product_category.updated_by IS '更新人';

-- 商品
CREATE TABLE product (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  name varchar(200) NOT NULL,
  slug varchar(120) NOT NULL,
  summary text NOT NULL DEFAULT ''::text,
  status varchar(20) NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  detail_html text NOT NULL DEFAULT ''::text,
  version bigint NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  created_by bigint,
  updated_by bigint,
  category_id bigint NOT NULL,
  home_featured boolean NOT NULL DEFAULT false,
  home_sort integer NOT NULL DEFAULT 0,
  CONSTRAINT product_home_sort_check CHECK ((home_sort >= 0)),
  CONSTRAINT product_pkey PRIMARY KEY (id),
  CONSTRAINT product_status_check CHECK (status IN ('draft', 'published', 'offline')),
  CONSTRAINT product_version_check CHECK ((version > 0))
);
CREATE INDEX product_category_active ON product USING btree (category_id) WHERE (deleted_at IS NULL);
CREATE INDEX product_home_featured ON product USING btree (home_sort, id) WHERE (home_featured AND ((status)::text = 'published'::text) AND (deleted_at IS NULL));
CREATE UNIQUE INDEX product_slug_active ON product USING btree (slug) WHERE (deleted_at IS NULL);
CREATE INDEX product_status_updated ON product USING btree (status, updated_at DESC);
COMMENT ON TABLE product IS '商品';
COMMENT ON COLUMN product.id IS '商品标识';
COMMENT ON COLUMN product.name IS '名称';
COMMENT ON COLUMN product.slug IS '官网地址标识';
COMMENT ON COLUMN product.summary IS '摘要';
COMMENT ON COLUMN product.status IS '状态:draft=草稿 published=已发布 offline=已下架';
COMMENT ON COLUMN product.updated_at IS '更新时间';
COMMENT ON COLUMN product.detail_html IS '商品详情';
COMMENT ON COLUMN product.version IS '内容版本';
COMMENT ON COLUMN product.created_at IS '创建时间';
COMMENT ON COLUMN product.deleted_at IS '删除时间';
COMMENT ON COLUMN product.created_by IS '创建人';
COMMENT ON COLUMN product.updated_by IS '更新人';
COMMENT ON COLUMN product.category_id IS '逻辑指向 product_category.id';
COMMENT ON COLUMN product.home_featured IS '首页推荐开关';
COMMENT ON COLUMN product.home_sort IS '首页展示顺序';

-- 商品规格维度
CREATE TABLE product_spec (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  product_id bigint NOT NULL,
  key text NOT NULL,
  name text NOT NULL,
  sort integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  created_by bigint,
  updated_by bigint,
  CONSTRAINT product_spec_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX product_spec_key ON product_spec USING btree (product_id, key) WHERE (deleted_at IS NULL);
COMMENT ON TABLE product_spec IS '商品规格维度';
COMMENT ON COLUMN product_spec.id IS '商品规格维度标识';
COMMENT ON COLUMN product_spec.product_id IS '逻辑指向 product.id';
COMMENT ON COLUMN product_spec.key IS '业务标识';
COMMENT ON COLUMN product_spec.name IS '名称';
COMMENT ON COLUMN product_spec.sort IS '展示顺序';
COMMENT ON COLUMN product_spec.created_at IS '创建时间';
COMMENT ON COLUMN product_spec.updated_at IS '更新时间';
COMMENT ON COLUMN product_spec.deleted_at IS '删除时间';
COMMENT ON COLUMN product_spec.created_by IS '创建人';
COMMENT ON COLUMN product_spec.updated_by IS '更新人';

-- 商品规格值
CREATE TABLE product_spec_value (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  product_id bigint NOT NULL,
  spec_id bigint NOT NULL,
  key text NOT NULL,
  name text NOT NULL,
  sort integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  created_by bigint,
  updated_by bigint,
  CONSTRAINT product_spec_value_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX product_spec_value_key ON product_spec_value USING btree (product_id, key) WHERE (deleted_at IS NULL);
CREATE INDEX product_spec_value_spec ON product_spec_value USING btree (spec_id) WHERE (deleted_at IS NULL);
COMMENT ON TABLE product_spec_value IS '商品规格值';
COMMENT ON COLUMN product_spec_value.id IS '商品规格值标识';
COMMENT ON COLUMN product_spec_value.product_id IS '逻辑指向 product.id';
COMMENT ON COLUMN product_spec_value.spec_id IS '逻辑指向 product_spec.id';
COMMENT ON COLUMN product_spec_value.key IS '业务标识';
COMMENT ON COLUMN product_spec_value.name IS '名称';
COMMENT ON COLUMN product_spec_value.sort IS '展示顺序';
COMMENT ON COLUMN product_spec_value.created_at IS '创建时间';
COMMENT ON COLUMN product_spec_value.updated_at IS '更新时间';
COMMENT ON COLUMN product_spec_value.deleted_at IS '删除时间';
COMMENT ON COLUMN product_spec_value.created_by IS '创建人';
COMMENT ON COLUMN product_spec_value.updated_by IS '更新人';

-- 商品销售规格组合
CREATE TABLE product_sku (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  product_id bigint NOT NULL,
  key text NOT NULL,
  currency text NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  sort integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  created_by bigint,
  updated_by bigint,
  CONSTRAINT product_sku_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX product_sku_key ON product_sku USING btree (product_id, key) WHERE (deleted_at IS NULL);
COMMENT ON TABLE product_sku IS '商品销售规格组合';
COMMENT ON COLUMN product_sku.id IS '商品销售规格组合标识';
COMMENT ON COLUMN product_sku.product_id IS '逻辑指向 product.id';
COMMENT ON COLUMN product_sku.key IS '业务标识';
COMMENT ON COLUMN product_sku.currency IS '币种代码';
COMMENT ON COLUMN product_sku.enabled IS '是否启用';
COMMENT ON COLUMN product_sku.sort IS '展示顺序';
COMMENT ON COLUMN product_sku.created_at IS '创建时间';
COMMENT ON COLUMN product_sku.updated_at IS '更新时间';
COMMENT ON COLUMN product_sku.deleted_at IS '删除时间';
COMMENT ON COLUMN product_sku.created_by IS '创建人';
COMMENT ON COLUMN product_sku.updated_by IS '更新人';

-- 商品规格组合与规格值关联
CREATE TABLE product_sku_value (
  sku_id bigint NOT NULL,
  spec_value_id bigint NOT NULL,
  CONSTRAINT product_sku_value_pkey PRIMARY KEY (sku_id, spec_value_id)
);
CREATE INDEX product_sku_value_value ON product_sku_value USING btree (spec_value_id);
COMMENT ON TABLE product_sku_value IS '商品规格组合与规格值关联';
COMMENT ON COLUMN product_sku_value.sku_id IS '逻辑指向 product_sku.id';
COMMENT ON COLUMN product_sku_value.spec_value_id IS '逻辑指向 product_spec_value.id';

-- 商品规格价格
CREATE TABLE product_sku_price (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  sku_id bigint NOT NULL,
  currency text NOT NULL,
  amount bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  created_by bigint,
  updated_by bigint,
  CONSTRAINT product_sku_price_amount_check CHECK ((amount >= 0)),
  CONSTRAINT product_sku_price_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX product_sku_price_currency ON product_sku_price USING btree (sku_id, currency) WHERE (deleted_at IS NULL);
COMMENT ON TABLE product_sku_price IS '商品规格价格';
COMMENT ON COLUMN product_sku_price.id IS '商品规格价格标识';
COMMENT ON COLUMN product_sku_price.sku_id IS '逻辑指向 product_sku.id';
COMMENT ON COLUMN product_sku_price.currency IS '币种代码';
COMMENT ON COLUMN product_sku_price.amount IS '最小货币单位金额';
COMMENT ON COLUMN product_sku_price.created_at IS '创建时间';
COMMENT ON COLUMN product_sku_price.updated_at IS '更新时间';
COMMENT ON COLUMN product_sku_price.deleted_at IS '删除时间';
COMMENT ON COLUMN product_sku_price.created_by IS '创建人';
COMMENT ON COLUMN product_sku_price.updated_by IS '更新人';

-- 新闻文章
CREATE TABLE article (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  title varchar(200) NOT NULL,
  slug varchar(120) NOT NULL,
  summary text NOT NULL DEFAULT ''::text,
  status varchar(20) NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  body_html text NOT NULL DEFAULT ''::text,
  cover_media_id bigint,
  first_published_at timestamptz,
  pinned boolean NOT NULL DEFAULT false,
  sort integer NOT NULL DEFAULT 0,
  version bigint NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  created_by bigint,
  updated_by bigint,
  CONSTRAINT article_pkey PRIMARY KEY (id),
  CONSTRAINT article_status_check CHECK (status IN ('draft', 'published', 'offline')),
  CONSTRAINT article_version_check CHECK ((version > 0))
);
CREATE INDEX article_public_sort ON article USING btree (pinned DESC, sort, first_published_at DESC NULLS LAST, id DESC) WHERE (((status)::text = 'published'::text) AND (deleted_at IS NULL));
CREATE UNIQUE INDEX article_slug_active ON article USING btree (slug) WHERE (deleted_at IS NULL);
COMMENT ON TABLE article IS '新闻文章';
COMMENT ON COLUMN article.id IS '新闻文章标识';
COMMENT ON COLUMN article.title IS '标题';
COMMENT ON COLUMN article.slug IS '官网地址标识';
COMMENT ON COLUMN article.summary IS '摘要';
COMMENT ON COLUMN article.status IS '状态:draft=草稿 published=已发布 offline=已下架';
COMMENT ON COLUMN article.updated_at IS '更新时间';
COMMENT ON COLUMN article.body_html IS '新闻正文';
COMMENT ON COLUMN article.cover_media_id IS '逻辑指向 media.id';
COMMENT ON COLUMN article.first_published_at IS '首次发布时间';
COMMENT ON COLUMN article.pinned IS '置顶';
COMMENT ON COLUMN article.sort IS '展示顺序';
COMMENT ON COLUMN article.version IS '内容版本';
COMMENT ON COLUMN article.created_at IS '创建时间';
COMMENT ON COLUMN article.deleted_at IS '删除时间';
COMMENT ON COLUMN article.created_by IS '创建人';
COMMENT ON COLUMN article.updated_by IS '更新人';

-- 站点品牌配置
CREATE TABLE site_config (
  id bigint NOT NULL,
  payload jsonb NOT NULL,
  version bigint NOT NULL DEFAULT 1,
  CONSTRAINT site_config_id_check CHECK ((id = 1)),
  CONSTRAINT site_config_pkey PRIMARY KEY (id),
  CONSTRAINT site_config_version_check CHECK (version > 0)
);
COMMENT ON TABLE site_config IS '站点品牌配置';
COMMENT ON COLUMN site_config.id IS '站点配置单例标识';
COMMENT ON COLUMN site_config.payload IS '品牌配置';
COMMENT ON COLUMN site_config.version IS '内容版本';

-- 官网 Banner 位置
CREATE TABLE banner_slot (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  code varchar(60) NOT NULL,
  name varchar(100) NOT NULL,
  CONSTRAINT banner_slot_code_key UNIQUE (code),
  CONSTRAINT banner_slot_pkey PRIMARY KEY (id)
);
COMMENT ON TABLE banner_slot IS '官网 Banner 位置';
COMMENT ON COLUMN banner_slot.id IS '官网 Banner 位置标识';
COMMENT ON COLUMN banner_slot.code IS '位置编码';
COMMENT ON COLUMN banner_slot.name IS '名称';

-- 官网 Banner 投放
CREATE TABLE banner (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  slot_code varchar(60) NOT NULL,
  title varchar(200) NOT NULL,
  subtitle text NOT NULL DEFAULT ''::text,
  image_media_id bigint NOT NULL,
  mobile_image_media_id bigint,
  cta_text varchar(100) NOT NULL DEFAULT ''::character varying,
  link text NOT NULL DEFAULT ''::text,
  sort integer NOT NULL DEFAULT 0,
  enabled boolean NOT NULL DEFAULT false,
  starts_at timestamptz,
  ends_at timestamptz,
  version bigint NOT NULL DEFAULT 1,
  created_by bigint NOT NULL,
  updated_by bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT banner_check CHECK (((ends_at IS NULL) OR (starts_at IS NULL) OR (ends_at > starts_at))),
  CONSTRAINT banner_pkey PRIMARY KEY (id),
  CONSTRAINT banner_version_check CHECK (version > 0)
);
CREATE INDEX banner_slot_active ON banner USING btree (slot_code, sort, id) WHERE (deleted_at IS NULL);
COMMENT ON TABLE banner IS '官网 Banner 投放';
COMMENT ON COLUMN banner.id IS '官网 Banner 投放标识';
COMMENT ON COLUMN banner.slot_code IS '逻辑指向 banner_slot.code';
COMMENT ON COLUMN banner.title IS '标题';
COMMENT ON COLUMN banner.subtitle IS '副标题';
COMMENT ON COLUMN banner.image_media_id IS '桌面图片素材标识';
COMMENT ON COLUMN banner.mobile_image_media_id IS '手机图片素材标识';
COMMENT ON COLUMN banner.cta_text IS '按钮文字';
COMMENT ON COLUMN banner.link IS '跳转地址';
COMMENT ON COLUMN banner.sort IS '展示顺序';
COMMENT ON COLUMN banner.enabled IS '是否启用';
COMMENT ON COLUMN banner.starts_at IS '投放开始时间';
COMMENT ON COLUMN banner.ends_at IS '投放结束时间';
COMMENT ON COLUMN banner.version IS '并发版本';
COMMENT ON COLUMN banner.created_by IS '创建人';
COMMENT ON COLUMN banner.updated_by IS '更新人';
COMMENT ON COLUMN banner.created_at IS '创建时间';
COMMENT ON COLUMN banner.updated_at IS '更新时间';
COMMENT ON COLUMN banner.deleted_at IS '删除时间';

-- 国家地区客服配置
CREATE TABLE whatsapp_group (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  name text NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  created_by bigint NOT NULL,
  updated_by bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT whatsapp_group_pkey PRIMARY KEY (id)
);
COMMENT ON TABLE whatsapp_group IS '国家地区客服配置';
COMMENT ON COLUMN whatsapp_group.id IS '国家地区客服配置标识';
COMMENT ON COLUMN whatsapp_group.name IS '配置名称';
COMMENT ON COLUMN whatsapp_group.enabled IS '配置启用状态';
COMMENT ON COLUMN whatsapp_group.created_by IS '创建人';
COMMENT ON COLUMN whatsapp_group.updated_by IS '更新人';
COMMENT ON COLUMN whatsapp_group.created_at IS '创建时间';
COMMENT ON COLUMN whatsapp_group.updated_at IS '更新时间';
COMMENT ON COLUMN whatsapp_group.deleted_at IS '删除时间';

-- 客服配置国家范围
CREATE TABLE whatsapp_group_country (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  group_id bigint NOT NULL,
  country_code varchar(2) NOT NULL,
  created_by bigint NOT NULL,
  updated_by bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT whatsapp_group_country_pkey PRIMARY KEY (id)
);
CREATE INDEX whatsapp_country_group ON whatsapp_group_country USING btree (group_id) WHERE (deleted_at IS NULL);
CREATE UNIQUE INDEX whatsapp_country_unique ON whatsapp_group_country USING btree (country_code) WHERE (deleted_at IS NULL);
COMMENT ON TABLE whatsapp_group_country IS '客服配置国家范围';
COMMENT ON COLUMN whatsapp_group_country.id IS '客服配置国家范围标识';
COMMENT ON COLUMN whatsapp_group_country.group_id IS '逻辑指向 whatsapp_group.id';
COMMENT ON COLUMN whatsapp_group_country.country_code IS '国家地区代码';
COMMENT ON COLUMN whatsapp_group_country.created_by IS '创建人';
COMMENT ON COLUMN whatsapp_group_country.updated_by IS '更新人';
COMMENT ON COLUMN whatsapp_group_country.created_at IS '创建时间';
COMMENT ON COLUMN whatsapp_group_country.updated_at IS '更新时间';
COMMENT ON COLUMN whatsapp_group_country.deleted_at IS '删除时间';

-- WhatsApp 客服账号
CREATE TABLE whatsapp_account (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  group_id bigint NOT NULL,
  name text NOT NULL,
  phone_number text NOT NULL,
  qr_code_media_id bigint NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  allocation_count bigint NOT NULL DEFAULT 0,
  last_allocated_at timestamptz,
  created_by bigint NOT NULL,
  updated_by bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT whatsapp_account_allocation_count_check CHECK ((allocation_count >= 0)),
  CONSTRAINT whatsapp_account_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX whatsapp_account_phone ON whatsapp_account USING btree (group_id, phone_number) WHERE (deleted_at IS NULL);
CREATE INDEX whatsapp_account_pool ON whatsapp_account USING btree (group_id, id) WHERE ((deleted_at IS NULL) AND enabled);
CREATE INDEX whatsapp_account_qr ON whatsapp_account USING btree (qr_code_media_id) WHERE (deleted_at IS NULL);
COMMENT ON TABLE whatsapp_account IS 'WhatsApp 客服账号';
COMMENT ON COLUMN whatsapp_account.id IS 'WhatsApp 客服账号标识';
COMMENT ON COLUMN whatsapp_account.group_id IS '逻辑指向 whatsapp_group.id';
COMMENT ON COLUMN whatsapp_account.name IS '客服名称';
COMMENT ON COLUMN whatsapp_account.phone_number IS '国际电话号码';
COMMENT ON COLUMN whatsapp_account.qr_code_media_id IS '逻辑指向 media.id';
COMMENT ON COLUMN whatsapp_account.enabled IS '账号启用状态';
COMMENT ON COLUMN whatsapp_account.allocation_count IS '累计分配次数';
COMMENT ON COLUMN whatsapp_account.last_allocated_at IS '最近分配时间';
COMMENT ON COLUMN whatsapp_account.created_by IS '创建人';
COMMENT ON COLUMN whatsapp_account.updated_by IS '更新人';
COMMENT ON COLUMN whatsapp_account.created_at IS '创建时间';
COMMENT ON COLUMN whatsapp_account.updated_at IS '更新时间';
COMMENT ON COLUMN whatsapp_account.deleted_at IS '删除时间';

-- 默认客服配置设置
CREATE TABLE whatsapp_routing_setting (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  singleton_key boolean NOT NULL DEFAULT true,
  default_group_id bigint,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  updated_by bigint NOT NULL,
  CONSTRAINT whatsapp_routing_setting_pkey PRIMARY KEY (id),
  CONSTRAINT whatsapp_routing_setting_singleton_key_check CHECK (singleton_key),
  CONSTRAINT whatsapp_routing_setting_singleton_key_key UNIQUE (singleton_key)
);
COMMENT ON TABLE whatsapp_routing_setting IS '默认客服配置设置';
COMMENT ON COLUMN whatsapp_routing_setting.id IS '默认客服配置设置标识';
COMMENT ON COLUMN whatsapp_routing_setting.singleton_key IS '单例标识';
COMMENT ON COLUMN whatsapp_routing_setting.default_group_id IS '逻辑指向 whatsapp_group.id';
COMMENT ON COLUMN whatsapp_routing_setting.created_at IS '创建时间';
COMMENT ON COLUMN whatsapp_routing_setting.updated_at IS '更新时间';
COMMENT ON COLUMN whatsapp_routing_setting.updated_by IS '更新人';

-- 客服账号轮询位置
CREATE TABLE whatsapp_cursor (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  group_id bigint NOT NULL,
  last_account_id bigint,
  allocation_count bigint NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT whatsapp_cursor_allocation_count_check CHECK ((allocation_count >= 0)),
  CONSTRAINT whatsapp_cursor_group_id_key UNIQUE (group_id),
  CONSTRAINT whatsapp_cursor_pkey PRIMARY KEY (id)
);
COMMENT ON TABLE whatsapp_cursor IS '客服账号轮询位置';
COMMENT ON COLUMN whatsapp_cursor.id IS '客服账号轮询位置标识';
COMMENT ON COLUMN whatsapp_cursor.group_id IS '逻辑指向 whatsapp_group.id';
COMMENT ON COLUMN whatsapp_cursor.last_account_id IS '最近分配账号的历史位置';
COMMENT ON COLUMN whatsapp_cursor.allocation_count IS '配置累计分配次数';
COMMENT ON COLUMN whatsapp_cursor.created_at IS '创建时间';
COMMENT ON COLUMN whatsapp_cursor.updated_at IS '更新时间';

-- 问答配置
CREATE TABLE faq (
  id bigint GENERATED ALWAYS AS IDENTITY NOT NULL,
  question text NOT NULL,
  answer text NOT NULL,
  sort integer NOT NULL DEFAULT 0,
  enabled boolean NOT NULL DEFAULT false,
  version bigint NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  created_by bigint,
  updated_by bigint,
  CONSTRAINT faq_answer_check CHECK ((btrim(answer) <> ''::text)),
  CONSTRAINT faq_pkey PRIMARY KEY (id),
  CONSTRAINT faq_question_check CHECK ((btrim(question) <> ''::text)),
  CONSTRAINT faq_version_check CHECK ((version > 0))
);
CREATE INDEX faq_active_sort ON faq USING btree (sort, id) WHERE (deleted_at IS NULL);
CREATE INDEX faq_public_sort ON faq USING btree (sort, id) WHERE ((deleted_at IS NULL) AND enabled);
COMMENT ON TABLE faq IS '问答配置';
COMMENT ON COLUMN faq.id IS '问答标识';
COMMENT ON COLUMN faq.question IS '问题';
COMMENT ON COLUMN faq.answer IS '答案';
COMMENT ON COLUMN faq.sort IS '展示顺序';
COMMENT ON COLUMN faq.enabled IS '是否开启';
COMMENT ON COLUMN faq.version IS '内容版本';
COMMENT ON COLUMN faq.created_at IS '创建时间';
COMMENT ON COLUMN faq.updated_at IS '更新时间';
COMMENT ON COLUMN faq.deleted_at IS '删除时间';
COMMENT ON COLUMN faq.created_by IS '创建人';
COMMENT ON COLUMN faq.updated_by IS '更新人';

INSERT INTO site_identity(id) VALUES (1);
