-- Restore live accounts to the old JSON contract. Per-account IDs, versions and deleted history cannot be represented there.
INSERT INTO site_config(id,payload,version) VALUES(1,'{}'::jsonb,1) ON CONFLICT DO NOTHING;
UPDATE site_config SET payload=jsonb_set(payload,'{socialAccounts}',COALESCE((
  SELECT jsonb_agg(jsonb_build_object('name',s.name,'platform',s.platform,'link',s.link,
    'iconMediaId',s.icon_media_id::text,'icon',m.url,'enabled',s.enabled) ORDER BY s.sort,s.id)
  FROM social_account s JOIN media m ON m.id=s.icon_media_id WHERE s.deleted_at IS NULL
),'[]'::jsonb)),version=version+1 WHERE id=1;
DELETE FROM media_reference WHERE owner_type='site_config' AND owner_id='1' AND field='social_icon';
INSERT INTO media_reference(media_id,owner_type,owner_id,field,sort)
SELECT icon_media_id,'site_config','1','social_icon',min(sort) FROM social_account
WHERE deleted_at IS NULL GROUP BY icon_media_id ON CONFLICT DO NOTHING;
DELETE FROM media_reference WHERE owner_type='social_account';
DROP TABLE social_account;
COMMENT ON COLUMN media_reference.owner_type IS '引用对象类型:product=商品 article=文章 banner=Banner site_config=品牌配置 whatsapp_account=客服账号 product_category=商品分类';
COMMENT ON COLUMN media_reference.field IS '引用字段:detail=商品正文 image=图片 report=检测报告 body=文章正文 cover=封面 mobile_image=手机图片 logo=品牌图片 qr_code=客服二维码 social_icon=社交图标';
COMMENT ON COLUMN audit_log.target_type IS '操作对象类型:auth=认证 user=用户 role=角色 media=素材 product=商品 product_category=商品分类 article=新闻 banner=Banner site_config=品牌配置 whatsapp_group=客服配置 faq=问答';
