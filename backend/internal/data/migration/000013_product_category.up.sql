CREATE TABLE product_category (
 id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
 name text NOT NULL CHECK (btrim(name) <> ''),
 sort integer NOT NULL DEFAULT 0,
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 deleted_at timestamptz,
 created_by bigint,
 updated_by bigint
);
CREATE UNIQUE INDEX product_category_name_active ON product_category(lower(name)) WHERE deleted_at IS NULL;
ALTER TABLE product ADD COLUMN category_id bigint;
CREATE INDEX product_category_active ON product(category_id) WHERE deleted_at IS NULL;
-- Preserve existing category names; products without a category remain unassigned.
INSERT INTO product_category(name)
 SELECT min(btrim(category)) FROM product WHERE btrim(category) <> '' GROUP BY lower(btrim(category));
UPDATE product p SET category_id=c.id FROM product_category c WHERE lower(btrim(p.category))=lower(c.name);
COMMENT ON TABLE product_category IS '商品分类';
COMMENT ON COLUMN product_category.name IS '分类名称';
COMMENT ON COLUMN product_category.sort IS '展示顺序';
COMMENT ON COLUMN product_category.version IS '内容版本';
COMMENT ON COLUMN product.category_id IS '逻辑指向 product_category.id';
