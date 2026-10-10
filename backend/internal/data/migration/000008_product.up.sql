ALTER TABLE product DROP COLUMN images,DROP COLUMN specifications,DROP COLUMN description;
ALTER TABLE product ADD COLUMN detail_html text NOT NULL DEFAULT '';
ALTER TABLE product ADD COLUMN body_format text NOT NULL DEFAULT 'richtext' CHECK(body_format='richtext');
ALTER TABLE product ADD COLUMN version bigint NOT NULL DEFAULT 1 CHECK(version>0);
ALTER TABLE product ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE product ADD COLUMN deleted_at timestamptz;
ALTER TABLE product ADD COLUMN created_by bigint;
ALTER TABLE product ADD COLUMN updated_by bigint;
ALTER TABLE product DROP CONSTRAINT product_slug_key;
CREATE UNIQUE INDEX product_slug_active ON product(slug) WHERE deleted_at IS NULL;
CREATE TABLE product_spec (
 id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
 product_id bigint NOT NULL, key text NOT NULL, name text NOT NULL, sort integer NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,
 created_by bigint,updated_by bigint
);
CREATE UNIQUE INDEX product_spec_key ON product_spec(product_id,key) WHERE deleted_at IS NULL;
CREATE TABLE product_spec_value (
 id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
 product_id bigint NOT NULL,spec_id bigint NOT NULL,key text NOT NULL,name text NOT NULL,sort integer NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,
 created_by bigint,updated_by bigint
);
CREATE UNIQUE INDEX product_spec_value_key ON product_spec_value(product_id,key) WHERE deleted_at IS NULL;
CREATE INDEX product_spec_value_spec ON product_spec_value(spec_id) WHERE deleted_at IS NULL;
CREATE TABLE product_sku (
 id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
 product_id bigint NOT NULL,key text NOT NULL,currency text NOT NULL,enabled boolean NOT NULL DEFAULT true,sort integer NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,
 created_by bigint,updated_by bigint
);
CREATE UNIQUE INDEX product_sku_key ON product_sku(product_id,key) WHERE deleted_at IS NULL;
CREATE TABLE product_sku_value (
 sku_id bigint NOT NULL,spec_value_id bigint NOT NULL,PRIMARY KEY(sku_id,spec_value_id)
);
CREATE INDEX product_sku_value_value ON product_sku_value(spec_value_id);
CREATE TABLE product_sku_price (
 id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
 sku_id bigint NOT NULL,currency text NOT NULL,amount bigint NOT NULL CHECK(amount>=0),
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,
 created_by bigint,updated_by bigint
);
CREATE UNIQUE INDEX product_sku_price_currency ON product_sku_price(sku_id,currency) WHERE deleted_at IS NULL;
COMMENT ON COLUMN product.detail_html IS '商品详情';
COMMENT ON COLUMN product.body_format IS '正文格式:richtext=富文本';
COMMENT ON COLUMN product.version IS '内容版本';
COMMENT ON COLUMN product_sku_price.amount IS '最小货币单位金额';
COMMENT ON COLUMN product_sku_price.currency IS '币种代码';
COMMENT ON COLUMN product_spec.product_id IS '逻辑指向 product.id';
COMMENT ON COLUMN product_spec_value.spec_id IS '逻辑指向 product_spec.id';
COMMENT ON COLUMN product_sku_value.spec_value_id IS '逻辑指向 product_spec_value.id';
COMMENT ON COLUMN product_sku_price.sku_id IS '逻辑指向 product_sku.id';
