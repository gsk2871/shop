ALTER TABLE banner ADD COLUMN product_id bigint REFERENCES product(id) ON DELETE SET NULL;
COMMENT ON COLUMN banner.product_id IS '首页轮播图关联商品';
CREATE TABLE lead_popup_config (
 id smallint PRIMARY KEY CHECK (id=1),
 payload jsonb NOT NULL,
 version bigint NOT NULL DEFAULT 1 CHECK (version>0)
);
COMMENT ON TABLE lead_popup_config IS '进站留资弹窗配置';
COMMENT ON COLUMN lead_popup_config.id IS '单站点配置主键';
COMMENT ON COLUMN lead_popup_config.payload IS '弹窗开关及英文文案';
COMMENT ON COLUMN lead_popup_config.version IS '乐观锁版本';
CREATE TABLE contact_lead (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 contact_type text NOT NULL CHECK (contact_type IN ('email','whatsapp')),
 contact varchar(254) NOT NULL,
 first_submitted_at timestamptz NOT NULL DEFAULT now(),
 last_submitted_at timestamptz NOT NULL DEFAULT now(),
 submission_count bigint NOT NULL DEFAULT 1 CHECK (submission_count>0),
 status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','completed','invalid')),
 note varchar(2000) NOT NULL DEFAULT '',
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 UNIQUE(contact_type,contact)
);
CREATE INDEX contact_lead_recent ON contact_lead(last_submitted_at DESC,id DESC);
COMMENT ON TABLE contact_lead IS '官网邮箱及 WhatsApp 留资';
COMMENT ON COLUMN contact_lead.id IS '留资主键';
COMMENT ON COLUMN contact_lead.contact_type IS '联系方式类型';
COMMENT ON COLUMN contact_lead.contact IS '规范化联系方式';
COMMENT ON COLUMN contact_lead.first_submitted_at IS '首次提交时间';
COMMENT ON COLUMN contact_lead.last_submitted_at IS '最近提交时间';
COMMENT ON COLUMN contact_lead.submission_count IS '累计提交次数';
COMMENT ON COLUMN contact_lead.status IS 'pending 待跟进、completed 已完成、invalid 无效';
COMMENT ON COLUMN contact_lead.note IS '跟进备注';
COMMENT ON COLUMN contact_lead.version IS '乐观锁版本';
