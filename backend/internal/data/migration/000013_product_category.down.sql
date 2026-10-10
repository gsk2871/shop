-- Retain current category names in the legacy text column for rollback.
UPDATE product p SET category=c.name FROM product_category c WHERE p.category_id=c.id;
DROP INDEX IF EXISTS product_category_active;
ALTER TABLE product DROP COLUMN category_id;
DROP TABLE product_category;
