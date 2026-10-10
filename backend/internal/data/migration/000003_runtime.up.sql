-- Runtime identities and durable storage locations. No business data is moved.
CREATE TABLE site_identity (
 id bigint PRIMARY KEY CHECK (id = 1),
 site_id text NOT NULL DEFAULT gen_random_uuid()::text,
 created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO site_identity(id) VALUES(1);
COMMENT ON TABLE site_identity IS '站点身份';
COMMENT ON COLUMN site_identity.site_id IS '站点标识';
CREATE TABLE storage_root (
 id text PRIMARY KEY,
 path text NOT NULL UNIQUE,
 created_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE storage_root IS '媒体存储根目录';
COMMENT ON COLUMN storage_root.id IS '存储根标识';
COMMENT ON COLUMN storage_root.path IS '存储目录';
ALTER TABLE media ADD COLUMN storage_root_id text;
ALTER TABLE media ADD COLUMN storage_key text;
COMMENT ON COLUMN media.storage_root_id IS '逻辑指向 storage_root.id';
COMMENT ON COLUMN media.storage_key IS '文件标识';
