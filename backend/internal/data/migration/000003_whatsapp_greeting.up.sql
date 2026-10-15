ALTER TABLE whatsapp_account ADD COLUMN greeting text NOT NULL DEFAULT '';
COMMENT ON COLUMN whatsapp_account.greeting IS '问候语';
