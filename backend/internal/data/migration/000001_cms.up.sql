CREATE TABLE IF NOT EXISTS admin_account (id bigint PRIMARY KEY CHECK (id=1), username varchar(100) NOT NULL, password_hash text NOT NULL);
CREATE TABLE IF NOT EXISTS admin_session (token_hash varchar(64) PRIMARY KEY, expires_at timestamptz NOT NULL);
CREATE INDEX IF NOT EXISTS admin_session_expiry ON admin_session(expires_at);
CREATE TABLE IF NOT EXISTS site_config (id bigint PRIMARY KEY CHECK (id=1), payload jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS article (id bigserial PRIMARY KEY, title varchar(200) NOT NULL, slug varchar(120) NOT NULL UNIQUE, summary text NOT NULL DEFAULT '', body text NOT NULL DEFAULT '', cover text NOT NULL DEFAULT '', status varchar(20) NOT NULL CHECK(status IN ('draft','published','offline')), seo_title text NOT NULL DEFAULT '', seo_description text NOT NULL DEFAULT '', updated_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS media (id bigserial PRIMARY KEY, name text NOT NULL, url text NOT NULL, mime text NOT NULL, size bigint NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS audit_log (id bigserial PRIMARY KEY, action text NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
