CREATE TABLE faq (
 id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
 question text NOT NULL CHECK (btrim(question) <> ''),
 answer text NOT NULL CHECK (btrim(answer) <> ''),
 sort integer NOT NULL DEFAULT 0,
 enabled boolean NOT NULL DEFAULT false,
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 deleted_at timestamptz,
 created_by bigint,
 updated_by bigint
);
CREATE INDEX faq_active_sort ON faq(sort,id) WHERE deleted_at IS NULL;
CREATE INDEX faq_public_sort ON faq(sort,id) WHERE deleted_at IS NULL AND enabled;
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
