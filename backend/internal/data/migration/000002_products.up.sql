CREATE TABLE IF NOT EXISTS product (
 id bigserial PRIMARY KEY,
 name varchar(200) NOT NULL,
 slug varchar(120) NOT NULL UNIQUE,
 category varchar(200) NOT NULL DEFAULT '',
 aliases text NOT NULL DEFAULT '',
 cas varchar(200) NOT NULL DEFAULT '',
 formula varchar(200) NOT NULL DEFAULT '',
 molecular_weight varchar(200) NOT NULL DEFAULT '',
 sequence text NOT NULL DEFAULT '',
 purity varchar(200) NOT NULL DEFAULT '',
 specifications text NOT NULL DEFAULT '',
 storage text NOT NULL DEFAULT '',
 solubility text NOT NULL DEFAULT '',
 summary text NOT NULL DEFAULT '',
 description text NOT NULL DEFAULT '',
 images jsonb NOT NULL DEFAULT '[]'::jsonb CHECK(jsonb_typeof(images) = 'array'),
 status varchar(20) NOT NULL CHECK(status IN ('draft','published','offline')),
 seo_title varchar(200) NOT NULL DEFAULT '',
 seo_description text NOT NULL DEFAULT '',
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS product_status_updated ON product(status, updated_at DESC);
