ALTER TABLE media ADD COLUMN media_type text NOT NULL CHECK(media_type IN ('image','video','document'));
ALTER TABLE media ADD COLUMN width integer;
ALTER TABLE media ADD COLUMN height integer;
ALTER TABLE media ADD COLUMN available boolean NOT NULL DEFAULT true;
ALTER TABLE media ADD COLUMN cleanup_pending boolean NOT NULL DEFAULT false;
ALTER TABLE media ADD COLUMN uploaded_by bigint;
ALTER TABLE media ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE media ADD COLUMN deleted_at timestamptz;
ALTER TABLE media ALTER COLUMN storage_root_id SET NOT NULL;
ALTER TABLE media ALTER COLUMN storage_key SET NOT NULL;
CREATE UNIQUE INDEX media_storage_key ON media(storage_root_id,storage_key);
CREATE INDEX media_available_type ON media(media_type,id DESC) WHERE available AND deleted_at IS NULL;
CREATE TABLE media_reference (
 id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
 media_id bigint NOT NULL, owner_type text NOT NULL, owner_id text NOT NULL, field text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(media_id,owner_type,owner_id,field)
);
CREATE INDEX media_reference_owner ON media_reference(owner_type,owner_id);
COMMENT ON COLUMN media.media_type IS '媒体类型:image=图片 video=视频 document=文档';
COMMENT ON COLUMN media.available IS '素材可用状态';
COMMENT ON COLUMN media.cleanup_pending IS '等待文件清理';
COMMENT ON COLUMN media.uploaded_by IS '上传者';
COMMENT ON COLUMN media.width IS '宽度（像素）';
COMMENT ON COLUMN media.height IS '高度（像素）';
COMMENT ON COLUMN media_reference.media_id IS '逻辑指向 media.id';
COMMENT ON COLUMN media_reference.owner_type IS '引用对象类型';
COMMENT ON COLUMN media_reference.owner_id IS '引用对象标识';
