CREATE TABLE social_account (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name text NOT NULL,
  platform text NOT NULL CHECK (platform IN ('facebook','tiktok','youtube','x','instagram','linkedin','other')),
  link text NOT NULL,
  icon_media_id bigint NOT NULL,
  enabled boolean NOT NULL DEFAULT false,
  sort integer NOT NULL DEFAULT 0,
  version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
  created_by bigint NOT NULL DEFAULT -1,
  updated_by bigint NOT NULL DEFAULT -1,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX social_account_public_sort ON social_account(sort,id) WHERE deleted_at IS NULL AND enabled;
COMMENT ON TABLE social_account IS '社交账号';
COMMENT ON COLUMN social_account.id IS '社交账号标识';
COMMENT ON COLUMN social_account.name IS '账号名称';
COMMENT ON COLUMN social_account.platform IS '平台:facebook=Facebook tiktok=TikTok youtube=YouTube x=X instagram=Instagram linkedin=LinkedIn other=其他';
COMMENT ON COLUMN social_account.link IS '账号链接';
COMMENT ON COLUMN social_account.icon_media_id IS '逻辑指向 media.id';
COMMENT ON COLUMN social_account.enabled IS '开启状态';
COMMENT ON COLUMN social_account.sort IS '展示顺序';
COMMENT ON COLUMN social_account.version IS '内容版本';
COMMENT ON COLUMN social_account.created_by IS '创建人';
COMMENT ON COLUMN social_account.updated_by IS '更新人';
COMMENT ON COLUMN social_account.created_at IS '创建时间';
COMMENT ON COLUMN social_account.updated_at IS '更新时间';
COMMENT ON COLUMN social_account.deleted_at IS '删除时间';

-- Preserve account order and disabled entries from the previously deployed JSON storage.
INSERT INTO social_account(name,platform,link,icon_media_id,enabled,sort)
SELECT item->>'name', item->>'platform', item->>'link', (item->>'iconMediaId')::bigint,
       COALESCE((item->>'enabled')::boolean,false), (position-1)::integer
FROM site_config,
LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(payload->'socialAccounts')='array'
  THEN payload->'socialAccounts' ELSE '[]'::jsonb END) WITH ORDINALITY AS accounts(item,position)
WHERE site_config.id=1 ORDER BY position;
INSERT INTO media_reference(media_id,owner_type,owner_id,field,sort)
SELECT icon_media_id,'social_account',id::text,'icon',0 FROM social_account;
DELETE FROM media_reference WHERE owner_type='site_config' AND owner_id='1' AND field='social_icon';
UPDATE site_config SET payload=payload-'socialAccounts',version=version+1 WHERE payload ? 'socialAccounts';
COMMENT ON COLUMN media_reference.owner_type IS '引用对象类型:product=商品 article=文章 banner=Banner site_config=品牌配置 whatsapp_account=客服账号 product_category=商品分类 social_account=社交账号';
COMMENT ON COLUMN media_reference.field IS '引用字段:detail=商品正文 image=图片 report=检测报告 body=文章正文 cover=封面 mobile_image=手机图片 logo=品牌图片 qr_code=客服二维码 icon=社交图标';
COMMENT ON COLUMN audit_log.target_type IS '操作对象类型:auth=认证 user=用户 role=角色 media=素材 product=商品 product_category=商品分类 article=新闻 banner=Banner site_config=品牌配置 whatsapp_group=客服配置 faq=问答 social_account=社交账号';
