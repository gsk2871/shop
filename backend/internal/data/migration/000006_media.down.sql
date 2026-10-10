DROP TABLE media_reference;
DROP INDEX media_available_type;
DROP INDEX media_storage_key;
ALTER TABLE media ALTER COLUMN storage_key DROP NOT NULL;
ALTER TABLE media ALTER COLUMN storage_root_id DROP NOT NULL;
ALTER TABLE media DROP COLUMN deleted_at,DROP COLUMN updated_at,DROP COLUMN uploaded_by,DROP COLUMN cleanup_pending,DROP COLUMN available,DROP COLUMN height,DROP COLUMN width,DROP COLUMN media_type;
