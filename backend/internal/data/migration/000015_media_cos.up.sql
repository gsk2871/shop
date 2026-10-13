CREATE TABLE media_group (
 id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
 name text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 deleted_at timestamptz
);
CREATE UNIQUE INDEX media_group_name ON media_group(lower(name)) WHERE deleted_at IS NULL;
COMMENT ON TABLE media_group IS '图片目录';
COMMENT ON COLUMN media_group.name IS '目录名称';
ALTER TABLE media ADD COLUMN group_id bigint;
ALTER TABLE media ADD COLUMN storage_provider text NOT NULL DEFAULT 'local' CHECK(storage_provider IN ('local','cos'));
ALTER TABLE media ADD COLUMN status text NOT NULL DEFAULT 'ready' CHECK(status IN ('pending','ready','failed','canceled'));
ALTER TABLE media ADD COLUMN batch_id text;
ALTER TABLE media ADD COLUMN temporary_cleaned boolean NOT NULL DEFAULT false;
COMMENT ON COLUMN media.temporary_cleaned IS '临时图片清理完成状态';
CREATE INDEX media_group_list ON media(group_id,id DESC) WHERE available AND deleted_at IS NULL AND media_type='image';
COMMENT ON COLUMN media.group_id IS '逻辑指向 media_group.id，空值表示未分组';
COMMENT ON COLUMN media.storage_provider IS '存储方式:local=本地 cos=腾讯云对象存储';
COMMENT ON COLUMN media.status IS '上传状态:pending=待完成 ready=可用 failed=失败 canceled=已取消';
COMMENT ON COLUMN media.batch_id IS '逻辑指向 media_upload_batch.id';
CREATE TABLE media_upload_batch (
 id text PRIMARY KEY,
 uploader_id bigint NOT NULL,
 bucket text NOT NULL,
 region text NOT NULL,
 public_domain text NOT NULL,
 key_prefix text NOT NULL,
 max_bytes bigint NOT NULL CHECK(max_bytes>0),
 expires_at timestamptz NOT NULL,
 cleanup_after timestamptz NOT NULL,
 cleaned_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX media_batch ON media(batch_id) WHERE batch_id IS NOT NULL;
CREATE INDEX media_upload_cleanup ON media_upload_batch(cleanup_after) WHERE cleaned_at IS NULL;
COMMENT ON TABLE media_upload_batch IS 'COS 图片上传批次';
COMMENT ON COLUMN media_upload_batch.uploader_id IS '上传者';
COMMENT ON COLUMN media_upload_batch.bucket IS '存储桶';
COMMENT ON COLUMN media_upload_batch.region IS '存储地域';
COMMENT ON COLUMN media_upload_batch.public_domain IS '图片访问域名';
COMMENT ON COLUMN media_upload_batch.key_prefix IS '对象路径前缀';
COMMENT ON COLUMN media_upload_batch.max_bytes IS '图片大小上限（字节）';
COMMENT ON COLUMN media_upload_batch.expires_at IS '批次失效时间';
COMMENT ON COLUMN media_upload_batch.cleanup_after IS '允许清理时间';
COMMENT ON COLUMN media_upload_batch.cleaned_at IS '临时文件清理完成时间';
