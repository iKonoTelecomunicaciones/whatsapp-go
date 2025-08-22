-- v0 -> v1 (compatible with v1+): Add the initial schema for the whatsapp cloud database
-- transaction: sqlite-fkey-off
CREATE TABLE IF NOT EXISTS wb_application (
    waba_id TEXT NOT NULL,
    business_phone_id TEXT NOT NULL,
    name TEXT,
    admin_user TEXT,
    page_access_token TEXT,
    PRIMARY KEY (waba_id)
);
