ALTER TABLE product ADD COLUMN home_featured boolean NOT NULL DEFAULT false;
ALTER TABLE product ADD COLUMN home_sort integer NOT NULL DEFAULT 0 CHECK (home_sort >= 0);
COMMENT ON COLUMN product.home_featured IS '首页推荐开关';
COMMENT ON COLUMN product.home_sort IS '首页展示顺序';
CREATE INDEX product_home_featured ON product(home_sort,id) WHERE home_featured AND status='published' AND deleted_at IS NULL;
