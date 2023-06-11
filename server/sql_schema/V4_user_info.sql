ALTER TABLE users ADD COLUMN app_version text;
ALTER TABLE users ADD COLUMN last_seen_time integer;
ALTER TABLE users ADD COLUMN last_upload_time integer;
ALTER TABLE users ADD COLUMN last_download_time integer;