ALTER TABLE media_reference ADD COLUMN sort integer NOT NULL DEFAULT 0;
COMMENT ON COLUMN media_reference.sort IS '展示顺序';
