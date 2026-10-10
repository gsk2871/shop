CREATE TABLE whatsapp_group (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, name text NOT NULL, enabled boolean NOT NULL DEFAULT true,
 created_by bigint NOT NULL, updated_by bigint NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz
);
CREATE TABLE whatsapp_group_country (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, group_id bigint NOT NULL, country_code varchar(2) NOT NULL,
 created_by bigint NOT NULL, updated_by bigint NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz
);
CREATE UNIQUE INDEX whatsapp_country_unique ON whatsapp_group_country(country_code) WHERE deleted_at IS NULL;
CREATE INDEX whatsapp_country_group ON whatsapp_group_country(group_id) WHERE deleted_at IS NULL;
CREATE TABLE whatsapp_account (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, group_id bigint NOT NULL, name text NOT NULL, phone_number text NOT NULL,
 qr_code_media_id bigint NOT NULL, enabled boolean NOT NULL DEFAULT true,
 allocation_count bigint NOT NULL DEFAULT 0 CHECK(allocation_count>=0), last_allocated_at timestamptz,
 created_by bigint NOT NULL, updated_by bigint NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz
);
CREATE UNIQUE INDEX whatsapp_account_phone ON whatsapp_account(group_id,phone_number) WHERE deleted_at IS NULL;
CREATE INDEX whatsapp_account_pool ON whatsapp_account(group_id,id) WHERE deleted_at IS NULL AND enabled;
CREATE INDEX whatsapp_account_qr ON whatsapp_account(qr_code_media_id) WHERE deleted_at IS NULL;
CREATE TABLE whatsapp_routing_setting (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 singleton_key boolean NOT NULL DEFAULT true UNIQUE CHECK(singleton_key), default_group_id bigint,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), updated_by bigint NOT NULL
);
CREATE TABLE whatsapp_cursor (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, group_id bigint NOT NULL UNIQUE, last_account_id bigint,
 allocation_count bigint NOT NULL DEFAULT 0 CHECK(allocation_count>=0),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE whatsapp_group IS '国家地区客服配置';
COMMENT ON TABLE whatsapp_group_country IS '客服配置国家范围';
COMMENT ON TABLE whatsapp_account IS 'WhatsApp 客服账号';
COMMENT ON TABLE whatsapp_routing_setting IS '默认客服配置设置';
COMMENT ON TABLE whatsapp_cursor IS '客服账号轮询位置';
COMMENT ON COLUMN whatsapp_group.name IS '配置名称';
COMMENT ON COLUMN whatsapp_group.enabled IS '配置启用状态';
COMMENT ON COLUMN whatsapp_group_country.group_id IS '逻辑指向 whatsapp_group.id';
COMMENT ON COLUMN whatsapp_group_country.country_code IS '国家地区代码';
COMMENT ON COLUMN whatsapp_account.group_id IS '逻辑指向 whatsapp_group.id';
COMMENT ON COLUMN whatsapp_account.name IS '客服名称';
COMMENT ON COLUMN whatsapp_account.phone_number IS '国际电话号码';
COMMENT ON COLUMN whatsapp_account.qr_code_media_id IS '逻辑指向 media.id';
COMMENT ON COLUMN whatsapp_account.enabled IS '账号启用状态';
COMMENT ON COLUMN whatsapp_account.allocation_count IS '累计分配次数';
COMMENT ON COLUMN whatsapp_account.last_allocated_at IS '最近分配时间';
COMMENT ON COLUMN whatsapp_routing_setting.singleton_key IS '单例标识';
COMMENT ON COLUMN whatsapp_routing_setting.default_group_id IS '逻辑指向 whatsapp_group.id';
COMMENT ON COLUMN whatsapp_cursor.group_id IS '逻辑指向 whatsapp_group.id';
COMMENT ON COLUMN whatsapp_cursor.last_account_id IS '最近分配账号的历史位置';
COMMENT ON COLUMN whatsapp_cursor.allocation_count IS '配置累计分配次数';
