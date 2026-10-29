ALTER TABLE product ADD COLUMN purity_label text NOT NULL DEFAULT '';
ALTER TABLE product ADD COLUMN recommendation integer CHECK (recommendation BETWEEN 1 AND 5);
COMMENT ON COLUMN product.purity_label IS '商品纯度标签';
COMMENT ON COLUMN product.recommendation IS '商品推荐指数';

ALTER TABLE product_sku_price ADD COLUMN original_amount bigint;
-- Preserve existing prices without inventing discounts. No business version changes:
-- all existing price amounts retain their meaning.
UPDATE product_sku_price SET original_amount=amount WHERE original_amount IS NULL;
ALTER TABLE product_sku_price ALTER COLUMN original_amount SET NOT NULL;
ALTER TABLE product_sku_price ADD CONSTRAINT product_price_discount CHECK (original_amount >= amount);
COMMENT ON COLUMN product_sku_price.original_amount IS '原价最小货币单位金额';

COMMENT ON COLUMN media_reference.field IS '引用字段:detail=商品正文 image=图片 report=检测报告 body=文章正文 cover=封面 mobile_image=手机图片 logo=品牌图片 qr_code=客服二维码 icon=社交图标 background=网站背景图片';
