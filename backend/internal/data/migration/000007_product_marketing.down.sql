-- Removing original prices loses discount metadata; current sale prices are retained.
ALTER TABLE product_sku_price DROP CONSTRAINT IF EXISTS product_price_discount;
ALTER TABLE product_sku_price DROP COLUMN IF EXISTS original_amount;
ALTER TABLE product DROP COLUMN IF EXISTS recommendation;
ALTER TABLE product DROP COLUMN IF EXISTS purity_label;

COMMENT ON COLUMN media_reference.field IS '引用字段:detail=商品正文 image=图片 report=检测报告 body=文章正文 cover=封面 mobile_image=手机图片 logo=品牌图片 qr_code=客服二维码 icon=社交图标';
