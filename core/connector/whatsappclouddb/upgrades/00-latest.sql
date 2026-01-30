-- v0 -> v1 (compatible with v1+): Add the initial schema for the whatsapp cloud database
-- transaction: sqlite-fkey-off
ALTER TABLE IF EXISTS wb_application
RENAME COLUMN business_id TO waba_id;

ALTER TABLE IF EXISTS wb_application
RENAME COLUMN wb_phone_id TO business_phone_id;

CREATE TABLE IF NOT EXISTS wb_application (
    business_id TEXT NOT NULL,
    wb_phone_id TEXT NOT NULL,
    name TEXT,
    admin_user TEXT,
    page_access_token TEXT,
    PRIMARY KEY (business_id)
);