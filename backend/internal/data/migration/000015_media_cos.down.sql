-- Rollback requires exporting new COS records first; never silently orphan them.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM media WHERE storage_provider='cos') THEN
  RAISE EXCEPTION 'COS media exists; export or migrate it before rollback';
 END IF;
END $$;
DROP TABLE media_upload_batch;
DROP INDEX media_group_list;
DROP INDEX media_batch;
ALTER TABLE media DROP COLUMN temporary_cleaned, DROP COLUMN batch_id, DROP COLUMN status, DROP COLUMN storage_provider, DROP COLUMN group_id;
DROP TABLE media_group;
