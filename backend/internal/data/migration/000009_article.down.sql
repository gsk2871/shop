DROP INDEX article_public_sort;
DROP INDEX article_slug_active;
ALTER TABLE article ADD CONSTRAINT article_slug_key UNIQUE(slug);
ALTER TABLE article DROP COLUMN updated_by,DROP COLUMN created_by,DROP COLUMN deleted_at,DROP COLUMN created_at,DROP COLUMN version,DROP COLUMN sort,DROP COLUMN pinned,DROP COLUMN first_published_at,DROP COLUMN cover_media_id,DROP COLUMN body_format,DROP COLUMN body_html;
ALTER TABLE article ADD COLUMN body text NOT NULL DEFAULT '';
ALTER TABLE article ADD COLUMN cover text NOT NULL DEFAULT '';
