-- Administrative identities and grants. No legacy account conversion.
CREATE TABLE admin_role (
 id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
 code text NOT NULL, name text NOT NULL, is_super boolean NOT NULL DEFAULT false,
 enabled boolean NOT NULL DEFAULT true,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz,
 created_by bigint, updated_by bigint
);
CREATE UNIQUE INDEX admin_role_code ON admin_role(code) WHERE deleted_at IS NULL;
CREATE TABLE admin_permission (
 id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
 code text NOT NULL UNIQUE, parent_id bigint, name text NOT NULL,
 node_type text NOT NULL CHECK(node_type IN ('directory','menu','action')),
 route text, permission_code text UNIQUE, sort integer NOT NULL DEFAULT 0, visibility text,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX admin_permission_parent ON admin_permission(parent_id,sort);
CREATE TABLE admin_role_permission (
 role_id bigint NOT NULL, permission_id bigint NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(role_id,permission_id)
);
CREATE TABLE admin_user (
 id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
 username text NOT NULL, display_name text NOT NULL, password_hash text NOT NULL,
 role_id bigint NOT NULL, enabled boolean NOT NULL DEFAULT true,
 must_change_pwd boolean NOT NULL DEFAULT true, auth_version bigint NOT NULL DEFAULT 1,
 totp_key_id text, totp_cipher text, last_login_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz,
 created_by bigint, updated_by bigint
);
CREATE UNIQUE INDEX admin_user_username ON admin_user(username) WHERE deleted_at IS NULL;
CREATE INDEX admin_user_role ON admin_user(role_id) WHERE deleted_at IS NULL;
ALTER TABLE admin_session ADD COLUMN user_id bigint NOT NULL;
ALTER TABLE admin_session ADD COLUMN scope text NOT NULL CHECK(scope IN ('full','onboarding','password_change'));
ALTER TABLE admin_session ADD COLUMN auth_version bigint NOT NULL;
ALTER TABLE admin_session ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE admin_session ADD COLUMN verified_at timestamptz;
CREATE INDEX admin_session_user ON admin_session(user_id);
ALTER TABLE audit_log ADD COLUMN user_id bigint;
ALTER TABLE audit_log ADD COLUMN target_type text NOT NULL DEFAULT '';
ALTER TABLE audit_log ADD COLUMN target_id text NOT NULL DEFAULT '';
ALTER TABLE audit_log ADD COLUMN result text NOT NULL DEFAULT 'success' CHECK(result IN ('success','failure'));
ALTER TABLE audit_log ADD COLUMN request_id text NOT NULL DEFAULT '';
ALTER TABLE audit_log ADD COLUMN ip text NOT NULL DEFAULT '';
ALTER TABLE audit_log ADD COLUMN changes jsonb NOT NULL DEFAULT '{}'::jsonb;
CREATE INDEX audit_log_actor_time ON audit_log(user_id,created_at DESC);
COMMENT ON TABLE admin_user IS '后台用户';
COMMENT ON COLUMN admin_user.role_id IS '逻辑指向 admin_role.id';
COMMENT ON COLUMN admin_user.auth_version IS '认证版本';
COMMENT ON COLUMN admin_user.totp_key_id IS '密钥标识';
COMMENT ON COLUMN admin_user.totp_cipher IS '验证器密文';
COMMENT ON COLUMN admin_permission.parent_id IS '逻辑指向 admin_permission.id';
COMMENT ON COLUMN admin_permission.node_type IS '节点类型:directory=目录 menu=菜单 action=动作';
COMMENT ON COLUMN admin_session.scope IS '会话范围:full=完整 onboarding=首次设置 password_change=修改密码';
COMMENT ON COLUMN admin_session.user_id IS '逻辑指向 admin_user.id';
COMMENT ON COLUMN audit_log.result IS '结果:success=成功 failure=失败';
