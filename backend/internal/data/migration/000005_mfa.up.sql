CREATE TABLE admin_mfa_ticket (
 token_hash text PRIMARY KEY, user_id bigint NOT NULL, auth_version bigint NOT NULL,
 expires_at timestamptz NOT NULL, attempts integer NOT NULL DEFAULT 0,
 consumed_at timestamptz, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX admin_mfa_ticket_user ON admin_mfa_ticket(user_id);
CREATE INDEX admin_mfa_ticket_expiry ON admin_mfa_ticket(expires_at);
CREATE TABLE admin_totp_pending (
 user_id bigint PRIMARY KEY, auth_version bigint NOT NULL,
 key_id text NOT NULL, cipher text NOT NULL, expires_at timestamptz NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE admin_totp_used (
 user_id bigint NOT NULL, timestep bigint NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(user_id,timestep)
);
CREATE INDEX admin_totp_used_time ON admin_totp_used(created_at);
CREATE TABLE admin_rate_limit (
 key_hash text PRIMARY KEY, window_start timestamptz NOT NULL, attempts integer NOT NULL,
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX admin_rate_limit_time ON admin_rate_limit(updated_at);
COMMENT ON TABLE admin_mfa_ticket IS '后台登录票据';
COMMENT ON TABLE admin_totp_pending IS '待绑定验证器';
COMMENT ON TABLE admin_totp_used IS '已使用验证时间步';
COMMENT ON TABLE admin_rate_limit IS '后台安全限流';
COMMENT ON COLUMN admin_mfa_ticket.user_id IS '逻辑指向 admin_user.id';
COMMENT ON COLUMN admin_totp_used.timestep IS '验证时间步';
