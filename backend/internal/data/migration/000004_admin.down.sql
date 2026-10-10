DROP INDEX audit_log_actor_time;
ALTER TABLE audit_log DROP COLUMN changes, DROP COLUMN ip, DROP COLUMN request_id, DROP COLUMN result, DROP COLUMN target_id, DROP COLUMN target_type, DROP COLUMN user_id;
DROP INDEX admin_session_user;
ALTER TABLE admin_session DROP COLUMN verified_at, DROP COLUMN created_at, DROP COLUMN auth_version, DROP COLUMN scope, DROP COLUMN user_id;
DROP TABLE admin_role_permission;
DROP TABLE admin_permission;
DROP TABLE admin_user;
DROP TABLE admin_role;
